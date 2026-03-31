from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles

from api.agents import router as agents_router
from api.files import router as files_router
from api.pages import router as pages_router
from app_settings import STATIC_DIR

app = FastAPI(title="Binary Download Service")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(files_router)
app.include_router(agents_router)
app.include_router(pages_router)
app.mount("/", StaticFiles(directory=STATIC_DIR, html=True), name="static")
