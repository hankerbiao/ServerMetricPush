import os
import tempfile
import unittest
import uuid
from unittest.mock import patch

from fastapi.testclient import TestClient

from database import FileRecord, SessionLocal
from main import app, parse_filename


class UploadReplaceTests(unittest.TestCase):
    def setUp(self):
        self.client = TestClient(app)
        self.temp_dir = tempfile.TemporaryDirectory()
        self.filename = f"node_exporter-test-{uuid.uuid4().hex}-linux-amd64.tar.gz"

    def tearDown(self):
        self.client.close()
        db = SessionLocal()
        try:
            db.query(FileRecord).filter(FileRecord.filename == self.filename).delete()
            db.commit()
        finally:
            db.close()
        self.temp_dir.cleanup()

    def test_upload_same_filename_replaces_existing_record(self):
        with patch("main.UPLOAD_DIR", self.temp_dir.name):
            first = self.client.post(
                "/api/upload",
                files={"file": (self.filename, b"first-version", "application/octet-stream")},
            )
            self.assertEqual(first.status_code, 200)

            second = self.client.post(
                "/api/upload",
                files={"file": (self.filename, b"second-version", "application/octet-stream")},
            )

        self.assertEqual(second.status_code, 200)

        db = SessionLocal()
        try:
            rows = db.query(FileRecord).filter(FileRecord.filename == self.filename).all()
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0].file_size, len(b"second-version"))
            self.assertTrue(os.path.exists(rows[0].file_path))
        finally:
            db.close()

    def test_parse_filename_supports_hyphenated_platform_suffix(self):
        parsed = parse_filename("node_exporter-1.10.2-darwin-arm64.tar.gz")

        self.assertEqual(parsed["program"], "node_exporter")
        self.assertEqual(parsed["version"], "1.10.2")
        self.assertEqual(parsed["os"], "darwin")
        self.assertEqual(parsed["arch"], "arm64")

    def test_parse_filename_supports_dotted_platform_suffix(self):
        parsed = parse_filename("node_exporter-1.10.2.darwin-arm64.tar.gz")

        self.assertEqual(parsed["program"], "node_exporter")
        self.assertEqual(parsed["version"], "1.10.2")
        self.assertEqual(parsed["os"], "darwin")
        self.assertEqual(parsed["arch"], "arm64")


if __name__ == "__main__":
    unittest.main()
