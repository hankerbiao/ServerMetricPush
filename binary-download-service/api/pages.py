import os

from fastapi import APIRouter
from fastapi.responses import FileResponse as FastAPIFileResponse

from app_settings import STATIC_DIR

router = APIRouter()


@router.get("/agents")
def agents_page():
    return FastAPIFileResponse(os.path.join(STATIC_DIR, "agents.html"))


@router.get("/versions")
def versions_page():
    return FastAPIFileResponse(os.path.join(STATIC_DIR, "versions.html"))
