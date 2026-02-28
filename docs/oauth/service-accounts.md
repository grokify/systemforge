# Service Accounts

Service accounts enable JWT Bearer authentication (RFC 7523) for machine-to-machine communication.

## Overview

Unlike client credentials which use a shared secret, service accounts use public-key cryptography:

1. Service account has an RSA/EC key pair
2. Client signs a JWT with the private key
3. Server verifies using the public key
4. Access token is issued

## Schema

### Service Account

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `name` | string | Display name |
| `email` | string | Unique identifier |
| `organization_id` | UUID | Owning organization |
| `created_by` | UUID | Creating user |
| `allowed_scopes` | []string | Requestable scopes |
| `active` | bool | Account status |

### Key Pair

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `service_account_id` | UUID | Parent account |
| `key_id` | string | JWK `kid` |
| `key_type` | enum | `rsa` or `ec` |
| `algorithm` | enum | RS256, ES256, etc. |
| `public_key_pem` | string | PEM-encoded public key |
| `expires_at` | time | Key expiration |
| `active` | bool | Key status |

## Creating Service Accounts

### Create Account

```go
sa, err := client.ServiceAccount.Create().
    SetName("CI/CD Pipeline").
    SetEmail("cicd@myorg.serviceaccount.local").
    SetOrganizationID(orgID).
    SetCreatedBy(adminUserID).
    SetAllowedScopes([]string{"deploy:staging", "deploy:production"}).
    Save(ctx)
```

### Generate Key Pair

```go
import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
)

// Generate RSA key pair
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
    return err
}

// Encode public key as PEM
pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
if err != nil {
    return err
}

pubPEM := pem.EncodeToMemory(&pem.Block{
    Type:  "PUBLIC KEY",
    Bytes: pubBytes,
})

// Generate key ID
keyID := generateKeyID()

// Store public key
_, err = client.ServiceAccountKeyPair.Create().
    SetServiceAccountID(sa.ID).
    SetKeyID(keyID).
    SetKeyType("rsa").
    SetAlgorithm("RS256").
    SetPublicKeyPem(string(pubPEM)).
    SetExpiresAt(time.Now().Add(365 * 24 * time.Hour)).
    Save(ctx)

// Return private key to user (export as PEM or JSON)
privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
privPEM := pem.EncodeToMemory(&pem.Block{
    Type:  "RSA PRIVATE KEY",
    Bytes: privBytes,
})

return privPEM // User downloads this
```

## JWT Bearer Flow

### Step 1: Create Assertion

Client creates and signs a JWT:

```go
import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func createAssertion(privateKey *rsa.PrivateKey, email, audience, keyID string) (string, error) {
    now := time.Now()

    claims := jwt.MapClaims{
        "iss": email,                          // Service account email
        "sub": email,                          // Subject
        "aud": audience,                       // Token endpoint URL
        "iat": now.Unix(),                     // Issued at
        "exp": now.Add(5 * time.Minute).Unix(), // Expires (max 1 hour)
        "jti": uuid.New().String(),            // Unique ID
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = keyID

    return token.SignedString(privateKey)
}
```

### Step 2: Request Token

```bash
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&
assertion=eyJhbGciOiJSUzI1NiIs...&
scope=deploy:staging
```

### Step 3: Verify Assertion (Server)

```go
func verifyAssertion(ctx context.Context, assertion string) (*ent.ServiceAccount, error) {
    // Parse without verification first
    token, _ := jwt.Parse(assertion, nil)
    claims := token.Claims.(jwt.MapClaims)

    // Get issuer (service account email)
    issuer := claims["iss"].(string)

    // Find service account
    sa, err := client.ServiceAccount.Query().
        Where(serviceaccount.EmailEQ(issuer)).
        WithKeyPairs().
        Only(ctx)
    if err != nil {
        return nil, ErrInvalidAssertion
    }

    // Find matching key
    keyID := token.Header["kid"].(string)
    var publicKey *rsa.PublicKey
    for _, kp := range sa.Edges.KeyPairs {
        if kp.KeyID == keyID && kp.Active && !kp.Revoked {
            block, _ := pem.Decode([]byte(kp.PublicKeyPem))
            pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
            publicKey = pub.(*rsa.PublicKey)
            break
        }
    }

    if publicKey == nil {
        return nil, ErrKeyNotFound
    }

    // Verify signature
    _, err = jwt.Parse(assertion, func(token *jwt.Token) (interface{}, error) {
        return publicKey, nil
    })
    if err != nil {
        return nil, ErrInvalidSignature
    }

    return sa, nil
}
```

## Complete Client Example

```go
package main

import (
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "io/ioutil"
    "net/http"
    "net/url"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type ServiceAccountClient struct {
    email      string
    keyID      string
    privateKey *rsa.PrivateKey
    tokenURL   string
}

func NewServiceAccountClient(keyFile, email, keyID, tokenURL string) (*ServiceAccountClient, error) {
    keyPEM, err := ioutil.ReadFile(keyFile)
    if err != nil {
        return nil, err
    }

    block, _ := pem.Decode(keyPEM)
    privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
        return nil, err
    }

    return &ServiceAccountClient{
        email:      email,
        keyID:      keyID,
        privateKey: privateKey,
        tokenURL:   tokenURL,
    }, nil
}

func (c *ServiceAccountClient) GetToken(scopes []string) (string, error) {
    // Create assertion
    assertion, err := c.createAssertion()
    if err != nil {
        return "", err
    }

    // Request token
    resp, err := http.PostForm(c.tokenURL, url.Values{
        "grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
        "assertion":  {assertion},
        "scope":      {strings.Join(scopes, " ")},
    })
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        AccessToken string `json:"access_token"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return result.AccessToken, nil
}

func (c *ServiceAccountClient) createAssertion() (string, error) {
    now := time.Now()

    claims := jwt.MapClaims{
        "iss": c.email,
        "sub": c.email,
        "aud": c.tokenURL,
        "iat": now.Unix(),
        "exp": now.Add(5 * time.Minute).Unix(),
        "jti": uuid.New().String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = c.keyID

    return token.SignedString(c.privateKey)
}
```

## Key Rotation

```go
// 1. Generate new key pair
newKey, newPEM := generateKeyPair()

// 2. Add to service account
client.ServiceAccountKeyPair.Create().
    SetServiceAccountID(saID).
    SetKeyID(newKeyID).
    SetPublicKeyPem(newPEM).
    // ...

// 3. Update clients to use new key

// 4. Revoke old key
client.ServiceAccountKeyPair.UpdateOneID(oldKeyID).
    SetRevoked(true).
    SetRevokedAt(time.Now()).
    Save(ctx)
```
