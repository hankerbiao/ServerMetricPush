import os
import re
import shutil

from fastapi import HTTPException, UploadFile
from sqlalchemy.orm import Session

from app_settings import (
    INSTALL_SCRIPT_NAME,
    INSTALL_SCRIPT_PROGRAM,
    UNINSTALL_SCRIPT_NAME,
    UPLOAD_DIR,
)
from database import FileRecord

FILENAME_PATTERN = re.compile(
    r"^(?P<program>node_exporter|node-push-exporter)-(?P<version>.+?)[.-](?P<os>linux|darwin)-(?P<arch>amd64|arm64)(?P<ext>\.tar\.gz)?$"
)


def parse_filename(filename: str) -> dict:
    """Parse a release artifact filename into persisted fields."""
    if filename == INSTALL_SCRIPT_NAME or filename == UNINSTALL_SCRIPT_NAME:
        return {
            "program": INSTALL_SCRIPT_PROGRAM,
            "version": filename,
            "os": "script",
            "arch": "shell",
        }

    match = FILENAME_PATTERN.match(filename)
    if match:
        return {
            "program": match.group("program"),
            "version": match.group("version"),
            "os": match.group("os"),
            "arch": match.group("arch"),
        }

    if filename.startswith("node_exporter"):
        program = "node_exporter"
    elif filename.startswith("node-push-exporter"):
        program = "node-push-exporter"
    else:
        program = "unknown"

    return {
        "program": program,
        "version": filename,
        "os": "unknown",
        "arch": "unknown",
    }


def save_uploaded_file(upload: UploadFile, destination_dir: str = None) -> tuple[str, int]:
    if destination_dir is None:
        destination_dir = UPLOAD_DIR
    file_path = os.path.join(destination_dir, upload.filename)
    with open(file_path, "wb") as buffer:
        shutil.copyfileobj(upload.file, buffer)
    return file_path, os.path.getsize(file_path)


def replace_existing_file_record(db: Session, filename: str) -> None:
    old_record = db.query(FileRecord).filter(FileRecord.filename == filename).first()
    if old_record is None:
        return

    delete_physical_file(old_record.file_path)
    db.delete(old_record)
    db.flush()


def create_file_record(db: Session, filename: str, file_path: str, file_size: int) -> FileRecord:
    parsed = parse_filename(filename)
    file_record = FileRecord(
        filename=filename,
        program=parsed["program"],
        version=parsed["version"],
        os=parsed["os"],
        arch=parsed["arch"],
        file_path=file_path,
        file_size=file_size,
    )
    db.add(file_record)
    db.commit()
    db.refresh(file_record)
    return file_record


def delete_file_record(db: Session, file_id: int) -> None:
    file_record = db.query(FileRecord).filter(FileRecord.id == file_id).first()
    if file_record is None:
        raise HTTPException(status_code=404, detail="文件不存在")

    delete_physical_file(file_record.file_path)
    db.delete(file_record)
    db.commit()


def get_existing_file_or_404(db: Session, filename: str) -> FileRecord:
    file_record = db.query(FileRecord).filter(FileRecord.filename == filename).first()
    if file_record is None or not os.path.exists(file_record.file_path):
        raise HTTPException(status_code=404, detail="文件不存在")
    return file_record


def delete_physical_file(file_path: str) -> None:
    if os.path.exists(file_path):
        os.remove(file_path)
