import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / 'deploy.sh'
WRAPPER = r'''
hostname() { printf '%s\n' "${TEST_HOST:-ytst-back}"; }
flock() { :; }
go() {
  printf '%s\n' "$*" >> "$TEST_LOG"
  if [[ -n "${TEST_FAIL:-}" && "$*" == "$TEST_FAIL"* ]]; then return 17; fi
  if [[ "$1" == build ]]; then printf 'compiled' > "$4"; fi
}
export -f hostname flock go
exec bash ./deploy.sh
'''


class DeployTests(unittest.TestCase):
    def run_pipeline(self, *, fail='', changed=True, missing=False, host='ytst-back', link=False):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        root = Path(directory.name)
        shutil.copyfile(SCRIPT, root / 'deploy.sh')
        (root / 'myapp').write_text('old')
        (root / 'keep.env').write_text('unchanged')
        if link:
            (root / 'myapp').unlink()
            (root / 'myapp').symlink_to(root / 'keep.env')
        env = {**os.environ, 'YTST_DEPLOY_LOCK': str(root / 'lock'),
               'TEST_LOG': str(root / 'commands'), 'TEST_FAIL': fail, 'TEST_HOST': host}
        env.pop('CHANGED_FILES', None)
        if changed:
            env['CHANGED_FILES'] = str(root / 'changed')
            if not missing:
                (root / 'changed').write_text('go.mod\nserver.go\n')
        result = subprocess.run(['bash', '-c', WRAPPER], cwd=root, env=env,
                                capture_output=True, text=True, timeout=15)
        commands = (root / 'commands').read_text().splitlines() if (root / 'commands').exists() else []
        return root, result, commands

    def test_archive_build_without_git_keeps_settings_and_does_not_start_application(self):
        root, result, commands = self.run_pipeline()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((root / '.git').exists())
        self.assertEqual((root / 'myapp').read_text(), 'compiled')
        self.assertEqual((root / 'keep.env').read_text(), 'unchanged')
        self.assertEqual(commands[:4], ['version', 'mod download', 'mod verify', 'test -short ./...'])
        self.assertTrue(commands[-1].startswith('build -trimpath -o .ytst-build.'))
        self.assertFalse(list(root.glob('.ytst-build.*')))

    def test_legacy_relay_without_changed_files_is_supported(self):
        _, result, _ = self.run_pipeline(changed=False)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_preserves_previous_binary_and_settings(self):
        for operation in ('mod download', 'mod verify', 'test', 'build'):
            with self.subTest(operation=operation):
                root, result, _ = self.run_pipeline(fail=operation)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual((root / 'myapp').read_text(), 'old')
                self.assertEqual((root / 'keep.env').read_text(), 'unchanged')
                self.assertFalse(list(root.glob('.ytst-build.*')))

    def test_bad_input_wrong_machine_and_symbolic_binary_fail_before_go(self):
        for options in ({'missing': True}, {'host': 'another-machine'}, {'link': True}):
            with self.subTest(options=options):
                root, result, commands = self.run_pipeline(**options)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(commands, [])
                self.assertEqual((root / 'keep.env').read_text(), 'unchanged')


if __name__ == '__main__':
    unittest.main()
