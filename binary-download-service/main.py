import os
import re
import shutil
from fastapi import FastAPI, Depends, HTTPException, UploadFile, File
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from sqlalchemy.orm import Session
from typing import Optional

from database import get_db, FileRecord, Base, engine
from schemas import FileResponse, FileListResponse

app = FastAPI(title="Binary Download Service")

# 解析文件名
FILENAME_PATTERN = re.compile(
    r'^(node_exporter|node-push-exporter)-(.+?)-(linux|darwin)-(amd64|arm64)(\.tar\.gz)?$'
)

UPLOAD_DIR = os.path.join(os.path.dirname(__file__), "uploads")
os.makedirs(UPLOAD_DIR, exist_ok=True)


@app.get("/api/files", response_model=FileListResponse)
def list_files(program: Optional[str] = None, db: Session = Depends(get_db)):
    query = db.query(FileRecord)
    if program:
        query = query.filter(FileRecord.program == program)
    files = query.order_by(FileRecord.uploaded_at.desc()).all()
    return FileListResponse(files=[FileResponse.model_validate(f) for f in files])


@app.delete("/api/files/{file_id}")
def delete_file(file_id: int, db: Session = Depends(get_db)):
    file_record = db.query(FileRecord).filter(FileRecord.id == file_id).first()
    if not file_record:
        raise HTTPException(status_code=404, detail="文件不存在")

    # 删除物理文件
    if os.path.exists(file_record.file_path):
        os.remove(file_record.file_path)

    # 删除数据库记录
    db.delete(file_record)
    db.commit()

    return {"message": "文件删除成功"}


def parse_filename(filename: str) -> Optional[dict]:
    """解析文件名提取 program, version, os, arch"""
    match = FILENAME_PATTERN.match(filename)
    if not match:
        return None
    return {
        "program": match.group(1),
        "version": match.group(2),
        "os": match.group(3),
        "arch": match.group(4),
    }


@app.post("/api/upload")
async def upload_file(file: UploadFile = File(...), db: Session = Depends(get_db)):
    # 解析文件名
    parsed = parse_filename(file.filename)
    if not parsed:
        raise HTTPException(
            status_code=400,
            detail="文件名格式不正确，应为: program-version-os-arch.tar.gz 例如: node_exporter-1.8.1-linux-amd64.tar.gz"
        )

    # 保存文件
    file_path = os.path.join(UPLOAD_DIR, file.filename)

    # 如果文件已存在，先删除旧文件
    if os.path.exists(file_path):
        old_record = db.query(FileRecord).filter(FileRecord.filename == file.filename).first()
        if old_record:
            if os.path.exists(old_record.file_path):
                os.remove(old_record.file_path)
            db.delete(old_record)

    with open(file_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)

    file_size = os.path.getsize(file_path)

    # 保存到数据库
    file_record = FileRecord(
        filename=file.filename,
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

    return {
        "id": file_record.id,
        "filename": file_record.filename,
        "message": "文件上传成功"
    }


@app.get("/download/{filename}")
def download_file(filename: str, db: Session = Depends(get_db)):
    file_record = db.query(FileRecord).filter(FileRecord.filename == filename).first()
    if not file_record or not os.path.exists(file_record.file_path):
        raise HTTPException(status_code=404, detail="文件不存在")

    return FileResponse(
        path=file_record.file_path,
        filename=filename,
        media_type="application/octet-stream"
    )


# 挂载静态文件
app.mount("/", StaticFiles(directory=os.path.join(os.path.dirname(__file__), "static"), html=True), name="static")