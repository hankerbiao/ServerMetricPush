import unittest
from datetime import timedelta

from database import AgentRecord, localnow
from services.agent_service import serialize_agent


class AgentServiceTests(unittest.TestCase):
    def test_serialize_agent_marks_recent_healthy_agent_online(self):
        now = localnow()
        agent = AgentRecord(
            agent_id="agent-1",
            hostname="host-1",
            version="1.0.0",
            os="linux",
            arch="amd64",
            ip="127.0.0.1",
            status="online",
            last_error=None,
            pushgateway_url="http://pushgateway:9091",
            push_interval_seconds=30,
            node_exporter_port=9100,
            node_exporter_metrics_url="http://127.0.0.1:9100/metrics",
            node_exporter_up=True,
            push_fail_count=0,
            started_at=now,
            last_seen_at=now - timedelta(seconds=10),
            registered_at=now,
            updated_at=now,
        )

        result = serialize_agent(agent)

        self.assertTrue(result.online)
        self.assertEqual(result.status, "online")

    def test_serialize_agent_marks_stale_agent_offline(self):
        now = localnow()
        agent = AgentRecord(
            agent_id="agent-2",
            hostname="host-2",
            version="1.0.0",
            os="linux",
            arch="amd64",
            ip="127.0.0.1",
            status="online",
            last_error=None,
            pushgateway_url="http://pushgateway:9091",
            push_interval_seconds=30,
            node_exporter_port=9100,
            node_exporter_metrics_url="http://127.0.0.1:9100/metrics",
            node_exporter_up=True,
            push_fail_count=0,
            started_at=now,
            last_seen_at=now - timedelta(seconds=120),
            registered_at=now,
            updated_at=now,
        )

        result = serialize_agent(agent)

        self.assertFalse(result.online)
        self.assertEqual(result.status, "offline")


if __name__ == "__main__":
    unittest.main()
