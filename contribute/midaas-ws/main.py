import os
import pathlib
import json
from fastapi import FastAPI, Response, Request, Body
from pydantic import BaseModel


KEYNAME = os.environ.get("MIDAAS_KEYNAME", "test")
KEYVALUE = os.environ.get("MIDAAS_KEYVALUE", "test")
ZONES = os.environ.get("MIDAAS_ZONE", "dev.local,test.local")
ALL_ZONES = ZONES.split(",")

INFO = f"""
Informations about current midaas instance:
- keyname: {KEYNAME} - (MIDAAS_KEYNAME env var)
- keyvalue: {KEYVALUE} - (MIDAAS_KEYVALUE env var)

Availables zones are comma separated: {ZONES} (MIDAAS_ZONE env var)
"""
app = FastAPI()


class TTLDelete(BaseModel):
    keyname: str
    keyvalue: str


class TTLCreate(BaseModel):
    ttl: int
    keyname: str
    keyvalue: str


def check_TSIG(keyname, keyvalue):
    return keyname == KEYNAME and KEYVALUE == keyvalue


def create_zone(file):
    print(f"Creating zone on {file}")
    with open(file, "w+") as f:
        json.dump({}, f)


def create_zone_if_not_exist(file):
    p = pathlib.Path(file)
    if not p.exists():
        create_zone(file)
    else:
        # check if zone is not empty
        with open(file, "r") as f:
            content = f.read()
            if not content:
                create_zone(file)


def read_zone(file):
    with open(file, "r") as f:
        data = json.loads(f.read())
    return data


@app.get("/ws/{domaine}")
async def list_domain(request: Request, domaine):
    records = {}
    for zone in ALL_ZONES:
        if zone in domaine.strip().lower():
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file)
            records = read_zone(file)
            return records
    return records


@app.get("/healthz")
async def health():
    return {"status": "OK"}


@app.put("/ws/{domaine}/{type}/{valeur}")
def create(response: Response, request: Request, domaine: str, type: str, valeur: str, TTL: TTLCreate) -> dict:
    if not check_TSIG(keyname=TTL.keyname, keyvalue=TTL.keyvalue):
        return {"status": "ERROR", "message": "wrong credentials"}

    for zone in ALL_ZONES:
        if zone in domaine:
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file=file)
            data = read_zone(file=file)
            with open(file, "w+") as f:
                updated_data = data | {domaine: {"type": type,
                                                 "valeur": valeur,
                                                 "ttl": TTL.ttl}}
                json.dump(updated_data, f)
            return {"status": "OK"}
    return {"status": "ERROR", "message": "zone not available"}


@app.delete("/ws/{domaine}/{type}/{valeur}")
def delete(response: Response, request: Request, domaine: str, type: str, valeur: str, TTL: TTLDelete) -> dict:
    if not check_TSIG(keyname=TTL.keyname, keyvalue=TTL.keyvalue):
        return {"status": "ERROR", "message": "wrong credentials"}

    for zone in ALL_ZONES:
        if zone in domaine:
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file=file)
            data = read_zone(file=file)
            print(domaine, data)
            with open(file, "w") as f:
                if domaine in data:
                    data.pop(domaine)
                json.dump(data, f)
            return {"status": "OK"}
    return {"status": "ERROR", "message": "no domain"}
