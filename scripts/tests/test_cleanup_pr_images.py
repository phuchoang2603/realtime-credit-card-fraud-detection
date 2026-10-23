import base64
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location(
    "cleanup_pr_images", Path(__file__).resolve().parents[1] / "cleanup-pr-images.py"
)
cleanup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cleanup)


class ImageRetentionTests(unittest.TestCase):
    def test_only_unreferenced_pr_versions_are_deleted(self):
        for tags, protected, delete in (
            (["pr-sha"], set(), True),
            (["pr-sha"], {"pr-sha"}, False),
            (["pr-sha", "main-sha"], {"main-sha"}, False),
            (["pr-sha", "latest"], set(), False),
            (["other-sha"], set(), False),
            ([], set(), False),
        ):
            with self.subTest(tags=tags, protected=protected):
                version = {"metadata": {"container": {"tags": tags}}}
                self.assertEqual(cleanup.should_delete(version, {"pr-sha"}, protected), delete)

    @patch.object(cleanup, "api")
    def test_chart_pins_protect_the_matching_packages(self, api):
        api.side_effect = [
            {"content": base64.b64encode(text.encode()).decode()}
            for text in ('imageTag: gateway-sha\n', 'image:\n  tag: "fraud-sha"\n', 'global:\n  imageTag: connect-sha\n')
        ]
        pins = cleanup.production_pins("owner/repo", "main")
        self.assertEqual(pins[""], {"fraud-sha"})
        self.assertEqual(pins["/connect-debezium"], {"connect-sha"})
        for image in cleanup.IMAGES[1:6]:
            self.assertEqual(pins[image], {"gateway-sha"})

    @patch.object(cleanup, "api", return_value={"content": base64.b64encode(b"imageTag: []\n").decode()})
    def test_unreadable_pin_stops_cleanup(self, api):
        with self.assertRaises(RuntimeError):
            cleanup.production_pins("owner/repo", "main")

    @patch.object(cleanup, "api", side_effect=RuntimeError("API unavailable"))
    def test_api_failure_stops_cleanup(self, api):
        with self.assertRaises(RuntimeError):
            list(cleanup.paginate("/versions"))
