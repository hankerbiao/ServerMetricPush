# Binary Download Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个 FastAPI 服务，支持上传、存储、管理和下载 node_exporter/node-push-exporter 二进制文件，提供 Web 管理界面

**Architecture:** FastAPI + SQLite + 静态前端。单文件应用，数据库使用 SQLite 存储元数据，文件存储在本地 uploads/ 目录

**Tech Stack:** FastAPI, Pydantic, SQLAlchemy, SQLite, vanilla HTML/CSS/JS

---

## File Structure

```
binary-download-service/
├── main.py                 # FastAPI 应用入口，路由和静态文件
├── models.py               # SQLAlchemy 数据库模型
├── database.py             # SQLite 数据库初始化
├── schemas.py              # Pydantic 请求/响应模型
├── requirements.txt        # Python 依赖
├── uploads/                # 上传文件存储目录 (创建)
└── static/
    └── index.html          # 前端管理页面
```

---

## Implementation Tasks

### Task 1: 项目初始化和依赖

**Files:**
- Create: `binary-download-service/requirements.txt`
- Create: `binary-download-service/database.py`

- [ ] **Step 1: 创建 requirements.txt**

```txt
fastapi==0.109.0
uvicorn==0.27.0
sqlalchemy==2.0.25
pydantic==2.5.3
python-multipart==0.0.6
```

- [ ] **Step 2: 创建 database.py**

```python
import os
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import Column, Integer, String, DateTime
from datetime import datetime

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATABASE_URL = f"sqlite:///{os.path.join(BASE_DIR, 'files.db')}"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()

class FileRecord(Base):
    __tablename__ = "files"

    id = Column(Integer, primary_key=True, index=True)
    filename = Column(String, unique=True, nullable=False)
    program = Column(String, nullable=False)
    version = Column(String, nullable=False)
    os = Column(String, nullable=False)
    arch = Column(String, nullable=False)
    file_path = Column(String, nullable=False)
    file_size = Column(Integer, nullable=False)
    uploaded_at = Column(DateTime, default=datetime.utcnow)

Base.metadata.create_all(bind=engine)

def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
```

- [ ] **Step 3: 创建 uploads 目录**

Run: `mkdir -p binary-download-service/uploads`

- [ ] **Step 4: Commit**

```bash
mkdir -p binary-download-service
cd binary-download-service
cat > requirements.txt << 'EOF'
fastapi==0.109.0
uvicorn==0.27.0
sqlalchemy==2.0.25
pydantic==2.5.3
python-multipart==0.0.6
EOF

cat > database.py << 'EOF'
import os
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import Column, Integer, String, DateTime
from datetime import datetime

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATABASE_URL = f"sqlite:///{os.path.join(BASE_DIR, 'files.db')}"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()

class FileRecord(Base):
    __tablename__ = "files"

    id = Column(Integer, primary_key=True, index=True)
    filename = Column(String, unique=True, nullable=False)
    program = Column(String, nullable=False)
    version = Column(String, nullable=False)
    os = Column(String, nullable=False)
    arch = Column(String, nullable=False)
    file_path = Column(String, nullable=False)
    file_size = Column(Integer, nullable=False)
    uploaded_at = Column(DateTime, default=datetime.utcnow)

Base.metadata.create_all(bind=engine)

def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
EOF

mkdir -p uploads
git add .
git commit -m "chore: init project structure and database setup"
```

---

### Task 2: Pydantic Schemas

**Files:**
- Create: `binary-download-service/schemas.py`

- [ ] **Step 1: 创建 schemas.py**

```python
from pydantic import BaseModel
from datetime import datetime
from typing import Optional

class FileBase(BaseModel):
    filename: str
    program: str
    version: str
    os: str
    arch: str

class FileResponse(FileBase):
    id: int
    file_size: int
    uploaded_at: datetime

    class Config:
        from_attributes = True

class FileListResponse(BaseModel):
    files: list[FileResponse]
```

- [ ] **Step 2: Commit**

```bash
cd binary-download-service
cat > schemas.py << 'EOF'
from pydantic import BaseModel
from datetime import datetime

class FileResponse(BaseModel):
    id: int
    filename: str
    program: str
    version: str
    os: str
    arch: str
    file_size: int
    uploaded_at: datetime

    class Config:
        from_attributes = True

class FileListResponse(BaseModel):
    files: list[FileResponse]
EOF

git add schemas.py
git commit -m "feat: add Pydantic schemas"
```

---

### Task 3: API Routes - 获取文件列表和删除

**Files:**
- Create: `binary-download-service/main.py`

- [ ] **Step 1: 创建 main.py - 获取文件列表 API**

```python
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
```

- [ ] **Step 2: 测试获取文件列表**

Run: `cd binary-download-service && python -c "from main import app; print('Import OK')"`

- [ ] **Step 3: Commit**

```bash
cd binary-download-service
git add main.py
git commit -m "feat: add list and delete file APIs"
```

---

### Task 4: API Routes - 文件上传

**Files:**
- Modify: `binary-download-service/main.py`

- [ ] **Step 1: 添加上传 API 到 main.py**

在 `import` 后添加上传相关函数:

```python
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
```

- [ ] **Step 2: 测试导入**

Run: `cd binary-download-service && python -c "from main import parse_filename; print(parse_filename('node_exporter-1.8.1-linux-amd64.tar.gz'))"`

Expected: `{'program': 'node_exporter', 'version': '1.8.1', 'os': 'linux', 'arch': 'amd64'}`

- [ ] **Step 3: Commit**

```bash
cd binary-download-service
git add main.py
git commit -m "feat: add file upload API"
```

---

### Task 5: 文件下载 API

**Files:**
- Modify: `binary-download-service/main.py`

- [ ] **Step 1: 添加下载 API**

```python
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
```

- [ ] **Step 2: Commit**

```bash
cd binary-download-service
git add main.py
git commit -m "feat: add file download API"
```

---

### Task 6: 前端页面

**Files:**
- Create: `binary-download-service/static/index.html`

- [ ] **Step 1: 创建前端页面**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Binary Download Service</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 900px; margin: 0 auto; }
        h1 { text-align: center; margin-bottom: 30px; color: #333; }
        .program-section { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .program-title { font-size: 1.2em; font-weight: bold; margin-bottom: 15px; color: #2563eb; }
        .upload-area { border: 2px dashed #ddd; padding: 20px; text-align: center; border-radius: 4px; margin-bottom: 15px; }
        .upload-area:hover { border-color: #2563eb; }
        .file-input { display: none; }
        .upload-btn { background: #2563eb; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        .upload-btn:hover { background: #1d4ed8; }
        .file-list { width: 100%; border-collapse: collapse; }
        .file-list th, .file-list td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        .file-list th { background: #f9fafb; font-weight: 600; }
        .btn { padding: 6px 12px; border: none; border-radius: 4px; cursor: pointer; margin-right: 5px; }
        .btn-download { background: #10b981; color: white; }
        .btn-download:hover { background: #059669; }
        .btn-delete { background: #ef4444; color: white; }
        .btn-delete:hover { background: #dc2626; }
        .btn-copy { background: #6b7280; color: white; }
        .btn-copy:hover { background: #4b5563; }
        .empty { text-align: center; color: #9ca3af; padding: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Binary Download Service</h1>

        <div class="program-section">
            <div class="program-title">node_exporter</div>
            <div class="upload-area">
                <input type="file" id="file-node_exporter" class="file-input" onchange="uploadFile(this, 'node_exporter')">
                <button class="upload-btn" onclick="document.getElementById('file-node_exporter').click()">上传文件</button>
                <p style="margin-top: 10px; color: #666; font-size: 0.9em;">文件名格式: node_exporter-{version}-{os}-{arch}.tar.gz</p>
            </div>
            <table class="file-list">
                <thead>
                    <tr>
                        <th>版本</th>
                        <th>架构</th>
                        <th>大小</th>
                        <th>上传时间</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="list-node_exporter"></tbody>
            </table>
        </div>

        <div class="program-section">
            <div class="program-title">node-push-exporter</div>
            <div class="upload-area">
                <input type="file" id="file-node-push-exporter" class="file-input" onchange="uploadFile(this, 'node-push-exporter')">
                <button class="upload-btn" onclick="document.getElementById('file-node-push-exporter').click()">上传文件</button>
                <p style="margin-top: 10px; color: #666; font-size: 0.9em;">文件名格式: node-push-exporter-{version}-{os}-{arch}</p>
            </div>
            <table class="file-list">
                <thead>
                    <tr>
                        <th>版本</th>
                        <th>架构</th>
                        <th>大小</th>
                        <th>上传时间</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="list-node-push-exporter"></tbody>
            </table>
        </div>
    </div>

    <script>
        const API_BASE = '';

        async function loadFiles() {
            const resp = await fetch(`${API_BASE}/api/files`);
            const data = await resp.json();

            const nodeExporterFiles = data.files.filter(f => f.program === 'node_exporter');
            const nodePushExporterFiles = data.files.filter(f => f.program === 'node-push-exporter');

            renderFileList('list-node_exporter', nodeExporterFiles);
            renderFileList('list-node-push-exporter', nodePushExporterFiles);
        }

        function renderFileList(elementId, files) {
            const tbody = document.getElementById(elementId);
            if (files.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="empty">暂无文件</td></tr>';
                return;
            }

            tbody.innerHTML = files.map(f => `
                <tr>
                    <td>${f.version}</td>
                    <td>${f.os}-${f.arch}</td>
                    <td>${formatSize(f.file_size)}</td>
                    <td>${new Date(f.uploaded_at).toLocaleString('zh-CN')}</td>
                    <td>
                        <button class="btn btn-copy" onclick="copyLink('${f.filename}')">复制链接</button>
                        <a href="/download/${f.filename}" class="btn btn-download" download>下载</a>
                        <button class="btn btn-delete" onclick="deleteFile(${f.id}, '${f.filename}')">删除</button>
                    </td>
                </tr>
            `).join('');
        }

        function formatSize(bytes) {
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
            if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
            return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
        }

        function copyLink(filename) {
            const url = `${window.location.origin}/download/${filename}`;
            navigator.clipboard.writeText(url).then(() => {
                alert('下载链接已复制: ' + url);
            });
        }

        async function uploadFile(input, program) {
            const file = input.files[0];
            if (!file) return;

            const formData = new FormData();
            formData.append('file', file);

            try {
                const resp = await fetch(`${API_BASE}/api/upload`, {
                    method: 'POST',
                    body: formData
                });

                if (resp.ok) {
                    alert('上传成功');
                    loadFiles();
                } else {
                    const err = await resp.json();
                    alert('上传失败: ' + err.detail);
                }
            } catch (e) {
                alert('上传失败: ' + e.message);
            }

            input.value = '';
        }

        async function deleteFile(id, filename) {
            if (!confirm(`确定要删除 ${filename} 吗?`)) return;

            try {
                const resp = await fetch(`${API_BASE}/api/files/${id}`, { method: 'DELETE' });
                if (resp.ok) {
                    alert('删除成功');
                    loadFiles();
                } else {
                    const err = await resp.json();
                    alert('删除失败: ' + err.detail);
                }
            } catch (e) {
                alert('删除失败: ' + e.message);
            }
        }

        loadFiles();
    </script>
</body>
</html>
```

- [ ] **Step 2: 更新 main.py 添加静态文件**

```python
# 在 main.py 末尾添加
app.mount("/", StaticFiles(directory=os.path.join(os.path.dirname(__file__), "static"), html=True), name="static")
```

- [ ] **Step 3: Commit**

```bash
cd binary-download-service
mkdir -p static
cat > static/index.html << 'HTMLEOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Binary Download Service</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 900px; margin: 0 auto; }
        h1 { text-align: center; margin-bottom: 30px; color: #333; }
        .program-section { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .program-title { font-size: 1.2em; font-weight: bold; margin-bottom: 15px; color: #2563eb; }
        .upload-area { border: 2px dashed #ddd; padding: 20px; text-align: center; border-radius: 4px; margin-bottom: 15px; }
        .upload-area:hover { border-color: #2563eb; }
        .file-input { display: none; }
        .upload-btn { background: #2563eb; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        .upload-btn:hover { background: #1d4ed8; }
        .file-list { width: 100%; border-collapse: collapse; }
        .file-list th, .file-list td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        .file-list th { background: #f9fafb; font-weight: 600; }
        .btn { padding: 6px 12px; border: none; border-radius: 4px; cursor: pointer; margin-right: 5px; text-decoration: none; display: inline-block; font-size: 0.9em; }
        .btn-download { background: #10b981; color: white; }
        .btn-download:hover { background: #059669; }
        .btn-delete { background: #ef4444; color: white; }
        .btn-delete:hover { background: #dc2626; }
        .btn-copy { background: #6b7280; color: white; }
        .btn-copy:hover { background: #4b5563; }
        .empty { text-align: center; color: #9ca3af; padding: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Binary Download Service</h1>

        <div class="program-section">
            <div class="program-title">node_exporter</div>
            <div class="upload-area">
                <input type="file" id="file-node_exporter" class="file-input" onchange="uploadFile(this, 'node_exporter')">
                <button class="upload-btn" onclick="document.getElementById('file-node_exporter').click()">上传文件</button>
                <p style="margin-top: 10px; color: #666; font-size: 0.9em;">文件名格式: node_exporter-{version}-{os}-{arch}.tar.gz</p>
            </div>
            <table class="file-list">
                <thead>
                    <tr>
                        <th>版本</th>
                        <th>架构</th>
                        <th>大小</th>
                        <th>上传时间</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="list-node_exporter"></tbody>
            </table>
        </div>

        <div class="program-section">
            <div class="program-title">node-push-exporter</div>
            <div class="upload-area">
                <input type="file" id="file-node-push-exporter" class="file-input" onchange="uploadFile(this, 'node-push-exporter')">
                <button class="upload-btn" onclick="document.getElementById('file-node-push-exporter').click()">上传文件</button>
                <p style="margin-top: 10px; color: #666; font-size: 0.9em;">文件名格式: node-push-exporter-{version}-{os}-{arch}</p>
            </div>
            <table class="file-list">
                <thead>
                    <tr>
                        <th>版本</th>
                        <th>架构</th>
                        <th>大小</th>
                        <th>上传时间</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="list-node-push-exporter"></tbody>
            </table>
        </div>
    </div>

    <script>
        const API_BASE = '';

        async function loadFiles() {
            const resp = await fetch(`${API_BASE}/api/files`);
            const data = await resp.json();

            const nodeExporterFiles = data.files.filter(f => f.program === 'node_exporter');
            const nodePushExporterFiles = data.files.filter(f => f.program === 'node-push-exporter');

            renderFileList('list-node_exporter', nodeExporterFiles);
            renderFileList('list-node-push-exporter', nodePushExporterFiles);
        }

        function renderFileList(elementId, files) {
            const tbody = document.getElementById(elementId);
            if (files.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="empty">暂无文件</td></tr>';
                return;
            }

            tbody.innerHTML = files.map(f => \`
                <tr>
                    <td>\${f.version}</td>
                    <td>\${f.os}-\${f.arch}</td>
                    <td>\${formatSize(f.file_size)}</td>
                    <td>\${new Date(f.uploaded_at).toLocaleString('zh-CN')}</td>
                    <td>
                        <button class="btn btn-copy" onclick="copyLink('\${f.filename}')">复制链接</button>
                        <a href="/download/\${f.filename}" class="btn btn-download" download>下载</a>
                        <button class="btn btn-delete" onclick="deleteFile(\${f.id}, '\${f.filename}')">删除</button>
                    </td>
                </tr>
            \`).join('');
        }

        function formatSize(bytes) {
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
            if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
            return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
        }

        function copyLink(filename) {
            const url = \`\${window.location.origin}/download/\${filename}\`;
            navigator.clipboard.writeText(url).then(() => {
                alert('下载链接已复制: ' + url);
            });
        }

        async function uploadFile(input, program) {
            const file = input.files[0];
            if (!file) return;

            const formData = new FormData();
            formData.append('file', file);

            try {
                const resp = await fetch(\`\${API_BASE}/api/upload\`, {
                    method: 'POST',
                    body: formData
                });

                if (resp.ok) {
                    alert('上传成功');
                    loadFiles();
                } else {
                    const err = await resp.json();
                    alert('上传失败: ' + err.detail);
                }
            } catch (e) {
                alert('上传失败: ' + e.message);
            }

            input.value = '';
        }

        async function deleteFile(id, filename) {
            if (!confirm(\`确定要删除 \${filename} 吗?\`)) return;

            try {
                const resp = await fetch(\`\${API_BASE}/api/files/\${id}\`, { method: 'DELETE' });
                if (resp.ok) {
                    alert('删除成功');
                    loadFiles();
                } else {
                    const err = await resp.json();
                    alert('删除失败: ' + err.detail);
                }
            } catch (e) {
                alert('删除失败: ' + e.message);
            }
        }

        loadFiles();
    </script>
</body>
</html>
HTMLEOF

# 更新 main.py 添加静态文件服务
# 在文件末尾添加:
# app.mount("/", StaticFiles(directory=os.path.join(os.path.dirname(__file__), "static"), html=True), name="static")

git add static/index.html main.py
git commit -m "feat: add frontend UI"
```

---

### Task 7: 启动脚本和测试

**Files:**
- Create: `binary-download-service/run.sh`

- [ ] **Step 1: 创建启动脚本**

```bash
#!/bin/bash
cd "$(dirname "$0")"
pip install -r requirements.txt -q
uvicorn main:app --host 0.0.0.0 --port 8080
```

- [ ] **Step 2: 测试服务启动**

Run: `cd binary-download-service && chmod +x run.sh && timeout 5 python -c "from main import app; print('OK')" && uvicorn main:app --host 0.0.0.0 --port 8080 &`

验证: 访问 http://localhost:8080 应该看到管理页面

- [ ] **Step 3: Commit**

```bash
cd binary-download-service
cat > run.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
pip install -r requirements.txt -q
uvicorn main:app --host 0.0.0.0 --port 8080
EOF
chmod +x run.sh

git add run.sh
git commit -m "chore: add startup script"
```

---

## Summary

完成所有任务后，你将拥有:
1. FastAPI 服务提供 REST API
2. SQLite 数据库存储文件元数据
3. 本地 uploads/ 目录存储实际文件
4. Web 前端管理界面
5. 可执行脚本快速启动服务

**启动命令:** `./run.sh` 或 `cd binary-download-service && uvicorn main:app --host 0.0.0.0 --port 8080`

**访问地址:** http://localhost:8080

---

Plan complete and saved to `docs/superpowers/plans/2026-03-27-binary-download-service.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**