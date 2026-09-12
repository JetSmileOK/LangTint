import hashlib
import json
from pathlib import Path
import shutil
import struct
import tempfile
import unittest
import zipfile
import release


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / 'repo'
        shutil.copytree(release.ROOT, self.root,
                        ignore=shutil.ignore_patterns('__pycache__', 'dist', 'stage', 'build', 'output', '*.exe', '.git'))
        self.c = release.config(self.root)
        b = bytearray(512)
        b[:2] = b'MZ'
        struct.pack_into('<I', b, 0x3c, 128)
        b[128:132] = b'PE\0\0'
        struct.pack_into('<H', b, 132, 0x8664)
        struct.pack_into('<H', b, 152, 0x20b)
        struct.pack_into('<H', b, 220, 2)
        self.binary = Path(self.tmp.name) / 'fixture.exe'
        self.binary.write_bytes(b)
        self.stage = Path(self.tmp.name) / 'stage'
        self.out = Path(self.tmp.name) / 'out'

    def make_stage(self):
        release.stage(self.binary, self.stage, 'a' * 40, self.root)

    def test_source_manifest_exact(self):
        release.verify_source(self.root)

    def test_assemble_is_integrity_gate(self):
        release.assemble(self.root)

    def test_changed_source_rejected(self):
        (self.root / 'source/logic.go').write_text('package wrong')
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_missing_source_rejected(self):
        (self.root / 'source/logic.go').unlink()
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_extra_source_rejected(self):
        (self.root / 'source/extra.go').write_text('package main')
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_config_rejects_version_injection(self):
        c = self.c.copy(); c['version'] = '1.7.0; echo BAD'
        (self.root / 'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_exe_path(self):
        c = self.c.copy(); c['exe'] = '../../other.exe'
        (self.root / 'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_floating_go(self):
        c = self.c.copy(); c['go_version'] = 'stable'
        (self.root / 'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_wrong_build_id(self):
        (self.root / 'packaging/BUILD_ID.txt').write_text('wrong')
        with self.assertRaises(ValueError): release.config(self.root)

    def test_invalid_commit_rejected(self):
        with self.assertRaises(ValueError): release.stage(self.binary, self.stage, 'main', self.root)

    def test_no_staging_overwrite(self):
        self.stage.mkdir()
        with self.assertRaises(ValueError): self.make_stage()

    def test_missing_manifest_rejected(self):
        (self.root / 'packaging/windows' / (self.c['exe'] + '.manifest')).unlink()
        with self.assertRaises(ValueError): self.make_stage()

    def test_missing_license_rejected(self):
        (self.root / 'LICENSE').unlink()
        with self.assertRaises(ValueError): self.make_stage()

    def test_not_pe_rejected(self):
        self.binary.write_bytes(b'not Windows')
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_invalid_offset_rejected(self):
        b = bytearray(self.binary.read_bytes()); struct.pack_into('<I', b, 0x3c, 100000)
        self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_signature_rejected(self):
        b = bytearray(self.binary.read_bytes()); b[128:132] = b'nope'; self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_arch_rejected(self):
        b = bytearray(self.binary.read_bytes()); struct.pack_into('<H', b, 132, 0x14c); self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_magic_rejected(self):
        b = bytearray(self.binary.read_bytes()); struct.pack_into('<H', b, 152, 0x10b); self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_subsystem_rejected(self):
        b = bytearray(self.binary.read_bytes()); struct.pack_into('<H', b, 220, 3); self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_installer_source_policy(self):
        release.verify_installer_source(self.root)

    def test_invalid_icon_rejected(self):
        (self.root / 'assets/LangTint.ico').write_bytes(b'bad')
        with self.assertRaises(ValueError): release.verify_installer_source(self.root)

    def test_installer_version_mismatch_rejected(self):
        p = self.root / 'packaging/installer/LangTint.iss'
        p.write_text(p.read_text().replace('#define MyAppVersion "1.7.0"', '#define MyAppVersion "9.9.9"'))
        with self.assertRaises(ValueError): release.verify_installer_source(self.root)

    def test_installer_admin_request_rejected(self):
        p = self.root / 'packaging/installer/LangTint.iss'
        p.write_text(p.read_text().replace('PrivilegesRequired=lowest', 'PrivilegesRequired=admin'))
        with self.assertRaises(ValueError): release.verify_installer_source(self.root)

    def test_installer_downloader_rejected(self):
        p = self.root / 'packaging/installer/LangTint.iss'
        p.write_text(p.read_text() + '\n; DownloadTemporaryFile\n')
        with self.assertRaises(ValueError): release.verify_installer_source(self.root)

    def test_stage_records_unsigned_state(self):
        self.make_stage()
        info = json.loads((self.stage / 'RELEASE.json').read_text())
        self.assertEqual(info['signing'], 'unsigned')
        self.assertEqual(info['commit'], 'a' * 40)

    def test_package_excludes_cmd_scripts(self):
        self.make_stage()
        z = release.package(self.stage, self.out, self.root)
        with zipfile.ZipFile(z) as p:
            names = set(p.namelist())
            self.assertFalse(any(n.lower().endswith('.cmd') for n in names))
            self.assertIn('LICENSE', names)
            self.assertIn('THIRD_PARTY_NOTICES.md', names)

    def test_package_rejects_extra_exe(self):
        self.make_stage(); (self.stage / 'old.exe').write_bytes(self.binary.read_bytes())
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_directory(self):
        self.make_stage(); (self.stage / 'unexpected').mkdir()
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_changed_binary(self):
        self.make_stage()
        with (self.stage / self.c['exe']).open('ab') as f: f.write(b'changed')
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_changed_version(self):
        self.make_stage(); p = self.stage / 'RELEASE.json'; i = json.loads(p.read_text()); i['version'] = '9.9.9'; p.write_text(json.dumps(i))
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_unknown_signing(self):
        self.make_stage(); p = self.stage / 'RELEASE.json'; i = json.loads(p.read_text()); i['signing'] = 'probably'; p.write_text(json.dumps(i))
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_false_signed_label(self):
        self.make_stage(); p = self.stage / 'RELEASE.json'; i = json.loads(p.read_text()); i['signing'] = 'signed'; p.write_text(json.dumps(i))
        with self.assertRaises(FileNotFoundError): release.package(self.stage, self.out, self.root)

    def test_package_rejects_stale_signature_record(self):
        self.make_stage(); p = self.stage / 'RELEASE.json'; i = json.loads(p.read_text()); i['signing'] = 'signed'; p.write_text(json.dumps(i))
        (self.stage / 'SIGNATURE.json').write_text(json.dumps({'status': 'Valid', 'sha256': '0' * 64}))
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_deterministic(self):
        self.make_stage()
        first = release.package(self.stage, self.out, self.root)
        second = release.package(self.stage, Path(self.tmp.name) / 'other', self.root)
        self.assertEqual(first.read_bytes(), second.read_bytes())

    def test_package_refuses_overwrite(self):
        self.make_stage(); release.package(self.stage, self.out, self.root)
        with self.assertRaises(ValueError): release.package(self.stage, self.out, self.root)

    def test_package_inner_hashes(self):
        self.make_stage(); p = release.package(self.stage, self.out, self.root)
        with zipfile.ZipFile(p) as z:
            for line in z.read('SHA256SUMS.txt').decode().splitlines():
                sha, name = line.split('  ', 1)
                self.assertEqual(hashlib.sha256(z.read(name)).hexdigest(), sha)


if __name__ == '__main__':
    unittest.main()

# Additional workflow/supply-chain invariants for the public installer release.
class WorkflowPolicyTests(unittest.TestCase):
    def setUp(self):
        self.root = release.ROOT

    def test_build_workflow_pins_inno_digest(self):
        text = (self.root / '.github/workflows/build.yml').read_text(encoding='utf-8')
        self.assertIn('0362a383ed217d4c4239b5933866dd96d3eb2102737da92f80f6057a4b40df2f', text)
        self.assertIn('Get-AuthenticodeSignature', text)
        self.assertIn("Pyrsys B\\.V\\.", text)

    def test_release_workflow_is_manual_draft(self):
        text = (self.root / '.github/workflows/release.yml').read_text(encoding='utf-8')
        self.assertIn('workflow_dispatch:', text)
        self.assertIn("github.ref == 'refs/heads/main'", text)
        self.assertIn('--draft --prerelease', text)
        self.assertNotIn('on:\n  push:', text)

    def test_release_workflow_builds_setup_and_portable(self):
        text = (self.root / '.github/workflows/release.yml').read_text(encoding='utf-8')
        self.assertIn('LangTint-Setup-x64.exe', text)
        self.assertIn('LangTint-v1.7.0-Windows10-x64-unsigned.zip', text)
        self.assertIn('go test -count=100', text)

    def test_active_workflows_have_no_signpath_secret(self):
        for p in (self.root / '.github/workflows').glob('*.yml'):
            text = p.read_text(encoding='utf-8')
            self.assertNotIn('SIGNPATH_API_TOKEN', text)
            self.assertNotIn('pull_request_target', text)

    def test_actions_are_full_sha_pinned(self):
        import re
        for p in list((self.root / '.github/workflows').glob('*.yml')) + [self.root / 'packaging/signpath-workflow.yml.example']:
            for action in re.findall(r'uses:\s*([^\s#]+)', p.read_text(encoding='utf-8')):
                self.assertRegex(action, r'^[A-Za-z0-9_./-]+@[0-9a-f]{40}$')

    def test_signpath_template_is_inactive(self):
        self.assertFalse((self.root / '.github/workflows/signpath.yml').exists())
        text = (self.root / 'packaging/signpath-workflow.yml.example').read_text(encoding='utf-8')
        self.assertIn('PREPARATION ONLY', text)
        self.assertIn('manual approval remains required', text)

    def test_signpath_artifact_uses_langtint_name(self):
        text = (self.root / 'packaging/signpath-artifact.xml').read_text(encoding='utf-8')
        self.assertIn('path="LangTint.exe"', text)
        self.assertNotIn('TaskbarLayoutTint', text)
