import os
import pathlib
import json
from fastapi import FastAPI, Response, Request, Body
from pydantic import BaseModel
import logging

logger = logging.getLogger('uvicorn.error')

KEYNAME = "ddns-key." + os.environ.get("MIDAAS_KEYNAME", "d1")
KEYVALUE = os.environ.get("MIDAAS_KEYVALUE", "test")
ZONES = os.environ.get("MIDAAS_ZONES", "d1.dev.local")
ALL_ZONES = ZONES.split(",")

app = FastAPI()


class TTLDelete(BaseModel):
    keyname: str
    keyvalue: str


class TTLCreate(BaseModel):
    ttl: int
    keyname: str
    keyvalue: str


def check_TSIG(keyname, keyvalue):
    if not keyname == KEYNAME and KEYVALUE == keyvalue:
        logger.info(f"Keyname or Keyvalue not match")
        logger.info(f"Keyname: {keyname} with {KEYNAME}")
        logger.info(f"Keyvalue: {keyname} with {KEYNAME}")
        return False
    return True


def create_zone(file):
    logger.info(f"Creating zone on {file}")
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
    logger.info(f"GET on url: {request.url}")
    records = {}
    for zone in ALL_ZONES:
        if zone in domaine.strip().lower():
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file)
            records = read_zone(file)
            logger.info(f"Zone content : {records}")
            return records
    return records


@app.get("/healthz")
async def health():
    return {"status": "OK"}


@app.put("/ws/{domaine}/{type}/{valeur}")
def create(response: Response, request: Request, domaine: str, type: str, valeur: str, TTL: TTLCreate) -> dict:
    logger.info(f"PUT on url: {request.url}")
    if not check_TSIG(keyname=TTL.keyname, keyvalue=TTL.keyvalue):
        return {"status": "ERROR", "message": "wrong credentials"}

    for zone in ALL_ZONES:
        if zone in domaine:
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file=file)
            data = read_zone(file=file)
            with open(file, "w+") as f:
                if type == "CNAME":
                    valeur += "."
                key = f"{domaine}./{type}/{valeur}"
                updated_data = data | {key: {"type": type,
                                             "value": valeur,
                                             "ttl": TTL.ttl}}
                json.dump(updated_data, f)
                logger.info(f"Zone content : {updated_data}")
            return {"status": "OK"}
    return {"status": "ERROR", "message": "zone not available"}


@app.delete("/ws/{domaine}/{type}/{valeur}")
def delete(response: Response, request: Request, domaine: str, type: str, valeur: str, TTL: TTLDelete) -> dict:
    logger.info(f"DELETE on url: {request.url}")
    if not check_TSIG(keyname=TTL.keyname, keyvalue=TTL.keyvalue):
        return {"status": "ERROR", "message": "wrong credentials"}

    for zone in ALL_ZONES:
        if zone in domaine:
            file = f"/tmp/{zone}"
            create_zone_if_not_exist(file=file)
            data = read_zone(file=file)
            with open(file, "w") as f:
                if type == "CNAME":
                    valeur += "."
                key = f"{domaine}./{type}/{valeur}"
                if key in data:
                    data.pop(key)
                json.dump(data, f)
                logger.info(f"Zone content : {data}")
            return {"status": "OK"}
    return {"status": "ERROR", "message": "no domain"}
