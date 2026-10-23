import importlib.util
import os
from pathlib import Path
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location(
    "cleanup_pr_images", Path(__file__).resolve().parents[1] / "cleanup-pr-images.py"
)
cleanup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cleanup)


class ImageRetentionTests(unittest.TestCase):
    def test_versions_with_matching_pr_tags_are_deleted(self):
        sha = "a" * 40
        pr_tag = f"pr-82-{sha}"
        for tags, delete in (
            ([pr_tag], True),
            ([pr_tag, f"pr-82-{'b' * 40}"], True),
            ([pr_tag, f"main-{sha}"], True),
            ([pr_tag, "latest"], True),
            ([pr_tag, f"pr-83-{sha}"], True),
            ([pr_tag, "release-v1"], True),
            ([f"pr-820-{sha}"], False),
            ([f"pr-8-{sha}"], False),
            ([f"main-{sha}"], False),
            ([sha[:7]], False),
            (["pr-82-invalid"], False),
            ([], False),
        ):
            with self.subTest(tags=tags):
                version = {"metadata": {"container": {"tags": tags}}}
                self.assertEqual(cleanup.should_delete(version, 82), delete)
        self.assertFalse(cleanup.should_delete({}, 82))

    @patch.object(cleanup, "paginate")
    @patch.object(cleanup, "api")
    def test_discovery_scopes_user_and_org_packages_to_repository(self, api, paginate):
        for account_type, namespace in (("User", "users"), ("Organization", "orgs")):
            with self.subTest(account_type=account_type):
                api.return_value = {"type": account_type}
                paginate.return_value = iter([
                    {"name": "repo/edge", "repository": {"full_name": "Owner/Repo"}},
                    {"name": "repo/new-service", "repository": {"full_name": "owner/repo"}},
                    {"name": "repo/unrelated", "repository": {"full_name": "owner/other"}},
                    {"name": "repo/unlinked", "repository": None},
                ])
                self.assertEqual(list(cleanup.repository_packages("owner/repo")), [
                    f"/{namespace}/owner/packages/container/repo%2Fedge/versions",
                    f"/{namespace}/owner/packages/container/repo%2Fnew-service/versions",
                ])
                paginate.assert_called_with(f"/{namespace}/owner/packages?package_type=container")

    @patch.object(cleanup, "api")
    def test_pagination_keeps_query_and_reads_all_pages(self, api):
        api.side_effect = [[{"id": n} for n in range(100)], [{"id": 100}]]
        result = list(cleanup.paginate("/packages?package_type=container"))
        self.assertEqual([item["id"] for item in result], list(range(101)))
        self.assertEqual([call.args[0] for call in api.call_args_list], [
            "/packages?package_type=container&per_page=100&page=1",
            "/packages?package_type=container&per_page=100&page=2",
        ])

    @patch.dict(os.environ, {"REPOSITORY": "owner/repo", "PR": "82"})
    @patch.object(cleanup, "repository_packages", return_value=iter(["/versions"]))
    @patch.object(cleanup, "api")
    def test_cleanup_reads_all_versions_before_deleting(self, api, packages):
        def version(identifier, tag):
            return {"id": identifier, "metadata": {"container": {"tags": [tag]}}}

        pr_tag = "pr-82-" + "a" * 40
        first_page = [version(n, pr_tag) for n in range(100)]
        api.side_effect = [first_page, [version(100, "main-" + "b" * 40)]] + [None] * 100
        cleanup.main()
        calls = api.call_args_list
        self.assertEqual([call.args[0] for call in calls[:2]], [
            "/versions?per_page=100&page=1", "/versions?per_page=100&page=2",
        ])
        self.assertEqual([call.args[0] for call in calls[2:]], [f"/versions/{n}" for n in range(100)])
        for call in calls[2:]:
            self.assertEqual(call.kwargs, {"method": "DELETE", "missing_ok": True})

    @patch.object(cleanup, "api", side_effect=RuntimeError("API unavailable"))
    def test_api_failure_stops_cleanup(self, api):
        with self.assertRaises(RuntimeError):
            list(cleanup.paginate("/versions"))
