"""Execute the actual release PowerShell blocks without contacting GitHub.

Requires Git and PowerShell 7 (pwsh), both present on our hosted CI images.
Git uses disposable real repositories; gh is replaced by a recording stub.
The runner preamble/exit-code epilogue are intentionally part of every test.
"""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / '.github/workflows/release.yml'


def run_block(name):
    """Read one literal run block; reject missing/ambiguous step names."""
    text = WORKFLOW.read_text(encoding='utf-8')
    marker = '      - name: ' + name + '\n'
    if text.count(marker) != 1:
        raise AssertionError('Expected one workflow step: ' + name)
    step = text.split(marker, 1)[1].split('\n      - ', 1)[0]
    return textwrap.dedent(step.split('        run: |\n', 1)[1])


class ReleaseWorkflowTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.pwsh = shutil.which('pwsh')
        if not cls.pwsh or not shutil.which('git'):
            raise RuntimeError('Release workflow tests require Git and PowerShell 7 (pwsh)')

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='langtint-workflow-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        for name in ('tools/release.py', 'packaging/release.json', 'packaging/BUILD_ID.txt'):
            dst = self.root / name
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, dst)
        self.git('init', '-q')
        self.git('commit', '--allow-empty', '-qm', 'old')
        self.old_sha = self.git('rev-parse', 'HEAD')
        self.git('commit', '--allow-empty', '-qm', 'current')
        self.sha = self.git('rev-parse', 'HEAD')
        self.env = dict(os.environ, REQUESTED_TAG='v1.7.1', GITHUB_SHA=self.sha,
                        GITHUB_REPOSITORY='example/LangTint', PYTHONUTF8='1')

    def git(self, *args):
        return subprocess.run(
            ['git', '-c', 'user.name=Workflow test', '-c', 'user.email=test@example.invalid',
             *args], cwd=self.root, check=True, capture_output=True, text=True,
            encoding='utf-8', timeout=20).stdout.strip()

    def execute(self, name, *, native_errors=False, setup=''):
        return self.execute_script(setup + '\n' + run_block(name), native_errors=native_errors)

    def execute_script(self, body, *, native_errors=False):
        script = self.root / 'step.ps1'
        script.write_text(
            "$ErrorActionPreference = 'Stop'\n"
            '$PSNativeCommandUseErrorActionPreference = $' + str(native_errors).lower() + '\n'
            + body + '\n'
            + "if (Test-Path -LiteralPath variable:\\LASTEXITCODE) { exit $LASTEXITCODE }\n",
            encoding='utf-8')
        return subprocess.run([self.pwsh, '-NoLogo', '-NoProfile', '-NonInteractive',
                               '-Command', ". './step.ps1'"], cwd=self.root, env=self.env,
                              capture_output=True, text=True, encoding='utf-8',
                              errors='replace', timeout=30)

    def assert_result(self, result, success):
        output = result.stdout + result.stderr
        if success:
            self.assertEqual(result.returncode, 0, output)
        else:
            self.assertNotEqual(result.returncode, 0, output)

    def validate(self, success):
        for native in (False, True):
            with self.subTest(native_errors=native):
                self.assert_result(self.execute('Validate requested version',
                                                native_errors=native), success)

    def test_missing_tag_is_success(self):
        self.validate(True)

    def test_lightweight_tag_at_current_commit(self):
        self.git('tag', 'v1.7.1')
        self.validate(True)

    def test_annotated_tag_at_current_commit(self):
        self.git('tag', '-a', 'v1.7.1', '-m', 'candidate')
        self.validate(True)

    def test_lightweight_tag_at_other_commit_is_rejected(self):
        self.git('tag', 'v1.7.1', self.old_sha)
        self.validate(False)

    def test_annotated_tag_at_other_commit_is_rejected(self):
        self.git('tag', '-a', 'v1.7.1', self.old_sha, '-m', 'old candidate')
        self.validate(False)

    def test_branch_with_same_name_does_not_shadow_tag(self):
        self.git('tag', 'v1.7.1')
        self.git('branch', 'v1.7.1', self.old_sha)
        self.validate(True)

    def test_non_commit_tag_is_rejected(self):
        blob = self.git('hash-object', '-w', 'packaging/release.json')
        self.git('tag', 'v1.7.1', blob)
        self.validate(False)

    def test_wrong_or_unsafe_requested_tag_is_rejected(self):
        for tag in ('', 'v1.7.2', 'v1.7.*', '--help', 'v1.7.1; Write-Output bad'):
            with self.subTest(tag=tag):
                self.env['REQUESTED_TAG'] = tag
                self.validate(False)

    def test_missing_version_config_is_rejected(self):
        (self.root / 'packaging/release.json').unlink()
        self.validate(False)

    def test_missing_git_repository_is_rejected(self):
        (self.root / '.git').rename(self.root / 'saved-git')
        self.validate(False)

    def gh_stub(self, pages, *, api_exit=0, create_exit=0):
        self.env.update(TEST_RELEASE_PAGES=json.dumps(pages),
                        TEST_API_EXIT=str(api_exit), TEST_CREATE_EXIT=str(create_exit))
        return r'''
function gh {
    $argv = @($args)
    ConvertTo-Json -InputObject $argv -Compress | Add-Content gh-calls.jsonl
    if ($argv[0] -eq 'api') {
        $global:LASTEXITCODE = [int]$env:TEST_API_EXIT
        if ($global:LASTEXITCODE -eq 0) { $env:TEST_RELEASE_PAGES }
    } elseif ($argv[0] -eq 'release' -and $argv[1] -eq 'create') {
        $global:LASTEXITCODE = [int]$env:TEST_CREATE_EXIT
    } elseif ($argv[0] -eq 'release' -and $argv[1] -eq 'view') {
        throw 'Unsafe existence probe: release view conflates missing release and API failure'
    } else {
        throw ('Unexpected gh command: ' + ($argv -join ' '))
    }
}
'''

    def create_draft(self, pages, *, success, api_exit=0, create_exit=0):
        result = self.execute('Create draft prerelease only', setup=self.gh_stub(
            pages, api_exit=api_exit, create_exit=create_exit))
        self.assert_result(result, success)
        return [json.loads(line) for line in
                (self.root / 'gh-calls.jsonl').read_text(encoding='utf-8-sig').splitlines()]

    def test_first_release_creates_draft_with_two_exact_assets(self):
        calls = self.create_draft([[]], success=True)
        self.assertEqual(calls[0], ['api', '--paginate', '--slurp',
                                    'repos/example/LangTint/releases?per_page=100'])
        self.assertEqual(calls[1], ['release', 'create', 'v1.7.1',
                                    'dist\\LangTint-Setup.exe', 'dist\\SHA256SUMS.txt',
                                    '--target', self.sha, '--draft', '--prerelease',
                                    '--title', 'LangTint v1.7.1', '--notes-file', 'release-notes.txt'])
        notes = (self.root / 'release-notes.txt').read_text(encoding='utf-8-sig')
        self.assertIn(self.sha, notes)
        self.assertIn('real Windows 10 desktop acceptance', notes)

    def test_unrelated_release_does_not_block(self):
        self.create_draft([[{'tag_name': 'v1.7.0'}]], success=True)

    def test_existing_release_is_never_overwritten(self):
        for draft in (False, True):
            with self.subTest(draft=draft):
                calls = self.create_draft([[{'tag_name': 'v1.7.1', 'draft': draft}]],
                                          success=False)
                self.assertTrue(all(c[0] == 'api' for c in calls))

    def test_existing_release_on_later_page_is_rejected(self):
        calls = self.create_draft([[{'tag_name': 'v1.7.0'}], [{'tag_name': 'v1.7.1'}]],
                                  success=False)
        self.assertEqual(len(calls), 1)

    def test_api_failure_never_attempts_creation(self):
        calls = self.create_draft([[]], success=False, api_exit=1)
        self.assertEqual(len(calls), 1)

    def test_invalid_api_json_never_attempts_creation(self):
        setup = self.gh_stub([[]])
        self.env['TEST_RELEASE_PAGES'] = 'invalid JSON'
        self.assert_result(self.execute('Create draft prerelease only', setup=setup), False)
        self.assertEqual(len((self.root / 'gh-calls.jsonl').read_text().splitlines()), 1)

    def test_create_failure_is_not_success(self):
        self.create_draft([[]], success=False, create_exit=1)

    def test_non_array_api_responses_never_attempt_creation(self):
        for response in (None, {}, [None], [{}]):
            with self.subTest(response=response):
                calls = self.create_draft(response, success=False)
                self.assertTrue(all(c[0] == 'api' for c in calls))

    def test_every_python_release_command_checks_its_exit_code(self):
        lines = WORKFLOW.read_text(encoding='utf-8').splitlines()
        checked = 0
        for index, line in enumerate(lines):
            if re.match(r'^          (?:\$version = )?python ', line):
                with self.subTest(command=line.strip()):
                    self.assertIn('if ($LASTEXITCODE -ne 0) { throw ', lines[index + 1])
                checked += 1
        self.assertEqual(checked, 11)

    def test_python_gate_failures_stop_before_following_commands(self):
        (self.root / 'source').mkdir()
        # Inert sentinels let the old workflow reach its end if it masks a failure.
        # They are never used as actual binary/package validation evidence.
        (self.root / 'internal-dist').mkdir()
        (self.root / 'internal-dist/LangTint-v1.7.1-Windows10-x64-unsigned.zip').touch()
        for step, count in (('Full source, installer and runtime gates', 5),
                            ('Stage deterministic portable ZIP for internal validation', 3)):
            for failure in range(1, count + 1):
                with self.subTest(step=step, failed_command=failure):
                    self.env['TEST_FAIL_COMMAND'] = str(failure)
                    calls_path = self.root / 'gate-calls.txt'
                    calls_path.write_text('')
                    result = self.execute(step, setup=r'''
$script:pythonCalls = 0
function python {
    $script:pythonCalls++
    "python $script:pythonCalls" | Add-Content gate-calls.txt
    $global:LASTEXITCODE = 0
    if ($script:pythonCalls -eq [int]$env:TEST_FAIL_COMMAND) { $global:LASTEXITCODE = 23 }
}
function go {
    'go' | Add-Content ../gate-calls.txt
    $global:LASTEXITCODE = 0
}
''')
                    self.assert_result(result, False)
                    self.assertEqual(calls_path.read_text(encoding='utf-8-sig').splitlines(),
                                     [f'python {i}' for i in range(1, failure + 1)])

    def test_all_release_powershell_blocks_parse(self):
        names = re.findall(r'^      - name: (.+)$', WORKFLOW.read_text(encoding='utf-8'), re.M)
        for index, name in enumerate(names):
            (self.root / f'syntax-{index}.ps1').write_text(run_block(name), encoding='utf-8')
        self.assert_result(self.execute_script(r'''
foreach ($file in Get-ChildItem syntax-*.ps1) {
    $tokens = $null
    $parseErrors = $null
    $null = [System.Management.Automation.Language.Parser]::ParseFile(
        $file.FullName, [ref]$tokens, [ref]$parseErrors)
    if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
}
'''), True)


if __name__ == '__main__':
    unittest.main()
