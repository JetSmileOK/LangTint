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
                        ignore=shutil.ignore_patterns('__pycache__', 'dist', 'stage', '*.exe', '.git'))
        self.c = release.config(self.root)
        b = bytearray(512)
        b[:2] = b'MZ'
        struct.pack_into('<I', b, 0x3c, 128)
        b[128:132] = b'PE\0\0'
        struct.pack_into('<H', b, 132, 0x8664)
        struct.pack_into('<H', b, 152, 0x20b)
        struct.pack_into('<H', b, 220, 2)
        self.binary = Path(self.tmp.name)/'fixture.exe'
        self.binary.write_bytes(b)
        self.stage = Path(self.tmp.name)/'stage'
        self.out = Path(self.tmp.name)/'out'

    def make_stage(self):
        release.stage(self.binary, self.stage, 'a'*40, self.root)

    def test_assemble_exact(self):
        release.assemble(self.root)
        release.verify_source(self.root)

    def test_missing_fragment(self):
        (self.root/'tools/internal/source_fragments/app_windows.go.part02').unlink()
        with self.assertRaises(ValueError): release.assemble(self.root)

    def test_tampered_fragment(self):
        (self.root/'tools/internal/source_fragments/app_windows.go.part01').write_text('corrupt')
        with self.assertRaises(ValueError): release.assemble(self.root)

    def test_extra_fragment(self):
        (self.root/'tools/internal/source_fragments/app_windows.go.part99').write_text('extra')
        with self.assertRaises(ValueError): release.assemble(self.root)

    def test_changed_source(self):
        (self.root/'source/logic.go').write_text('package wrong')
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_missing_source(self):
        (self.root/'source/logic.go').unlink()
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_extra_source(self):
        (self.root/'source/extra.go').write_text('package main')
        with self.assertRaises(ValueError): release.verify_source(self.root)

    def test_config_rejects_version_injection(self):
        c = self.c.copy(); c['version']='1.5.3; echo BAD'
        (self.root/'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_exe_path(self):
        c = self.c.copy(); c['exe']='../../other.exe'
        (self.root/'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_floating_go(self):
        c = self.c.copy(); c['go_version']='stable'
        (self.root/'packaging/release.json').write_text(json.dumps(c))
        with self.assertRaises(ValueError): release.config(self.root)

    def test_config_rejects_wrong_build(self):
        (self.root/'packaging/BUILD_ID.txt').write_text('wrong')
        with self.assertRaises(ValueError): release.config(self.root)

    def test_invalid_commit(self):
        with self.assertRaises(ValueError): release.stage(self.binary,self.stage,'main',self.root)

    def test_no_staging_overwrite(self):
        self.stage.mkdir()
        with self.assertRaises(ValueError): self.make_stage()

    def test_missing_manifest(self):
        (self.root/'packaging/windows'/(self.c['exe']+'.manifest')).unlink()
        with self.assertRaises(ValueError): self.make_stage()

    def test_missing_license(self):
        (self.root/'LICENSE').unlink()
        with self.assertRaises(ValueError): self.make_stage()

    def test_not_pe(self):
        self.binary.write_bytes(b'not Windows')
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_invalid_offset(self):
        b=bytearray(self.binary.read_bytes());struct.pack_into('<I',b,0x3c,100000)
        self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_signature(self):
        b=bytearray(self.binary.read_bytes());b[128:132]=b'nope';self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_arch(self):
        b=bytearray(self.binary.read_bytes());struct.pack_into('<H',b,132,0x14c);self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_magic(self):
        b=bytearray(self.binary.read_bytes());struct.pack_into('<H',b,152,0x10b);self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_pe_wrong_subsystem(self):
        b=bytearray(self.binary.read_bytes());struct.pack_into('<H',b,220,3);self.binary.write_bytes(b)
        with self.assertRaises(ValueError): release.check_pe(self.binary)

    def test_stage_is_unsigned(self):
        self.make_stage()
        info=json.loads((self.stage/'RELEASE.json').read_text())
        self.assertEqual(info['signing'],'unsigned')
        self.assertEqual(info['commit'],'a'*40)

    def test_pack_excludes_local_reports(self):
        (self.root/'my_private_report.txt').write_text('private information')
        self.make_stage()
        z=release.package(self.stage,self.out,self.root)
        with zipfile.ZipFile(z) as p:
            self.assertNotIn('my_private_report.txt',p.namelist())
            self.assertIn('LICENSE',p.namelist())
            self.assertIn('THIRD_PARTY_NOTICES.md',p.namelist())

    def test_pack_rejects_extra_exe(self):
        self.make_stage();(self.stage/'old.exe').write_bytes(self.binary.read_bytes())
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_directory(self):
        self.make_stage();(self.stage/'unexpected').mkdir()
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_changed_binary(self):
        self.make_stage()
        with (self.stage/self.c['exe']).open('ab') as f:f.write(b'changed')
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_changed_version(self):
        self.make_stage();p=self.stage/'RELEASE.json';i=json.loads(p.read_text());i['version']='9.9.9';p.write_text(json.dumps(i))
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_unknown_signing(self):
        self.make_stage();p=self.stage/'RELEASE.json';i=json.loads(p.read_text());i['signing']='probably signed';p.write_text(json.dumps(i))
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_false_signed_label(self):
        self.make_stage();p=self.stage/'RELEASE.json';i=json.loads(p.read_text());i['signing']='signed';p.write_text(json.dumps(i))
        with self.assertRaises(FileNotFoundError): release.package(self.stage,self.out,self.root)

    def test_pack_rejects_stale_signature_record(self):
        self.make_stage();p=self.stage/'RELEASE.json';i=json.loads(p.read_text());i['signing']='signed';p.write_text(json.dumps(i))
        (self.stage/'SIGNATURE.json').write_text(json.dumps({'status':'Valid','sha256':'0'*64}))
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_deterministic(self):
        self.make_stage()
        first=release.package(self.stage,self.out,self.root)
        second=release.package(self.stage,Path(self.tmp.name)/'other',self.root)
        self.assertEqual(first.read_bytes(),second.read_bytes())

    def test_pack_refuses_overwrite(self):
        self.make_stage();release.package(self.stage,self.out,self.root)
        with self.assertRaises(ValueError): release.package(self.stage,self.out,self.root)

    def test_pack_inner_hashes(self):
        self.make_stage();p=release.package(self.stage,self.out,self.root)
        with zipfile.ZipFile(p) as z:
            for line in z.read('SHA256SUMS.txt').decode().splitlines():
                sha,name=line.split('  ',1)
                self.assertEqual(hashlib.sha256(z.read(name)).hexdigest(),sha)

    def test_pack_outer_hash(self):
        self.make_stage();p=release.package(self.stage,self.out,self.root)
        sha,name=(self.out/'SHA256SUMS.txt').read_text().strip().split('  ',1)
        self.assertEqual(release.digest(p),sha)
        self.assertEqual(p.name,name)

    def test_active_workflows_no_signpath_secret(self):
        for p in (self.root/'.github/workflows').glob('*.yml'):
            text=p.read_text()
            self.assertNotIn('SIGNPATH_API_TOKEN',text)
            self.assertNotIn('pull_request_target',text)

    def test_actions_pinned(self):
        import re
        paths=list((self.root/'.github/workflows').glob('*.yml'))+[self.root/'packaging/signpath-workflow.yml.example']
        for p in paths:
            for action in re.findall(r'uses:\s*([^\s#]+)',p.read_text()):
                self.assertRegex(action,r'^[A-Za-z0-9_./-]+@[0-9a-f]{40}$')

    def test_release_is_draft(self):
        text=(self.root/'.github/workflows/release.yml').read_text()
        self.assertIn('--draft --prerelease',text)
        self.assertIn("github.ref == 'refs/heads/main'",text)
        self.assertNotIn('gh release upload',text)

    def test_signing_template_guards(self):
        text=(self.root/'packaging/signpath-workflow.yml.example').read_text()
        for required in ('environment: signing','SIGNPATH_READY','VERSIONINFO prerequisite','artifact-id','verify_signature.ps1'):
            self.assertIn(required,text)
        self.assertNotIn('continue-on-error',text)

    def test_signature_verifier_is_strict(self):
        text=(self.root/'tools/verify_signature.ps1').read_text()
        for required in ('Get-AuthenticodeSignature',"$sig.Status -ne 'Valid'",'Thumbprint','TimeStamperCertificate','ProductName','Get-FileHash'):
            self.assertIn(required,text)

if __name__ == '__main__': unittest.main()
