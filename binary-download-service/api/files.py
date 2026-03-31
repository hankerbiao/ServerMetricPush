from typing import Optional

from fastapi import APIRouter, Depends, File, UploadFile
from fastapi.responses import FileResponse as FastAPIFileResponse
from sqlalchemy.orm import Session

from database import FileRecord, get_db
from schemas import FileListResponse, FileResponse as FileRecordResponse
from services.file_service import (
    create_file_record,
    delete_file_record,
    get_existing_file_or_404,
    replace_existing_file_record,
    save_uploaded_file,
)

router = APIRouter()


@router.get("/api/files", response_model=FileListResponse)
def list_files(program: Optional[str] = None, db: Session = Depends(get_db)):
    query = db.query(FileRecord)
    if program:
        query = query.filter(FileRecord.program == program)
    files = query.order_by(FileRecord.uploaded_at.desc()).all()
    return FileListResponse(files=[FileRecordResponse.model_validate(f) for f in files])


@router.delete("/api/files/{file_id}")
def delete_file(file_id: int, db: Session = Depends(get_db)):
    delete_file_record(db, file_id)
    return {"message": "文件删除成功"}


@router.post("/api/upload")
async def upload_file(file: UploadFile = File(...), db: Session = Depends(get_db)):
    replace_existing_file_record(db, file.filename)
    file_path, file_size = save_uploaded_file(file)
    file_record = create_file_record(db, file.filename, file_path, file_size)
    return {
        "id": file_record.id,
        "filename": file_record.filename,
        "message": "文件上传成功",
    }


@router.get("/download/{filename}")
def download_file(filename: str, db: Session = Depends(get_db)):
    file_record = get_existing_file_or_404(db, filename)
    return FastAPIFileResponse(
        path=file_record.file_path,
        filename=filename,
        media_type="application/octet-stream",
    )
