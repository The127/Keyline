from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from jose import jwt, jwk
import httpx
from functools import lru_cache

OIDC_DISCOVERY_URL = "http://192.168.1.41:8081/oidc/keyline/.well-known/openid-configuration"
EXPECTED_AUDIENCE = "example"

app = FastAPI()
security = HTTPBearer()


@lru_cache
def get_oidc_config():
    """Fetch OIDC discovery and JWKS data once and cache it."""
    config = httpx.get(OIDC_DISCOVERY_URL, timeout=5).json()
    jwks_uri = config["jwks_uri"]
    jwks = httpx.get(jwks_uri, timeout=5).json()
    return {
        "issuer": config["issuer"],
        "jwks": {key["kid"]: key for key in jwks["keys"]},
    }


def verify_jwt(token: str):
    config = get_oidc_config()

    try:
        header = jwt.get_unverified_header(token)
        jwk_data = config["jwks"].get(header["kid"])
    except Exception as e:
        raise HTTPException(status_code=401, detail=f"Invalid token header: {e}")

    if not jwk_data:
        raise HTTPException(status_code=401, detail="Unknown key ID")

    key = jwk.construct(jwk_data)

    try:
        payload = jwt.decode(
            token,
            key,
            algorithms=[jwk_data.get("alg", "RS256")],
            issuer=config["issuer"],
            audience=EXPECTED_AUDIENCE,  # ✅ Enforce correct audience
        )
        return payload
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=401, detail="Token expired")
    except jwt.JWTClaimsError as e:
        raise HTTPException(status_code=401, detail=f"Invalid claims: {e}")
    except jwt.JWTError as e:
        raise HTTPException(status_code=401, detail=f"Invalid token: {e}")


def get_current_user(credentials: HTTPAuthorizationCredentials = Depends(security)):
    return verify_jwt(credentials.credentials)


def require_role(role: str):
    def checker(user: dict = Depends(get_current_user)):
        roles = user.get("roles") or user.get("realm_access", {}).get("roles", [])
        if role not in roles:
            raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Forbidden")
        return user
    return checker


@app.get("/public")
def public():
    return {"message": "This is a public endpoint"}


@app.get("/me")
def me(user: dict = Depends(get_current_user)):
    return {"subject": user.get("sub"), "claims": user}


@app.get("/subscriber")
def subscriber(user: dict = Depends(require_role("subscriber"))):
    return {"message": "Welcome, Keyline subscriber!"}
