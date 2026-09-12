from pathlib import Path
import re
import struct
import unittest
import release

ROOT = release.ROOT
ISS = ROOT / 'packaging/installer/LangTint.iss'
APP = ROOT / 'source/app_windows.go'
MODE = ROOT / 'source/mode.go'
MANIFEST = ROOT / 'packaging/windows/LangTint.exe.manifest'

class InstallerPolicyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.iss = ISS.read_text(encoding='utf-8')
        cls.app = APP.read_text(encoding='utf-8')
        cls.mode = MODE.read_text(encoding='utf-8')
        cls.manifest = MANIFEST.read_text(encoding='utf-8')

    def test_01_setup_single_exe_name(self): self.assertIn('OutputBaseFilename=LangTint-Setup-x64', self.iss)
    def test_02_current_user_install(self): self.assertIn('PrivilegesRequired=lowest', self.iss)
    def test_03_per_user_programs_path(self): self.assertIn(r'DefaultDirName={localappdata}\Programs\LangTint', self.iss)
    def test_04_x64_setup_binary(self): self.assertIn('SetupArchitecture=x64', self.iss)
    def test_05_true_x64_windows_only(self): self.assertIn('ArchitecturesAllowed=x64os', self.iss)
    def test_06_minimum_windows10_1607(self): self.assertIn('MinVersion=10.0.14393', self.iss)
    def test_07_no_admin_override(self): self.assertNotIn('PrivilegesRequired=admin', self.iss)
    def test_08_modern_dynamic_ui(self): self.assertIn('WizardStyle=modern dynamic', self.iss)
    def test_09_setup_icon(self): self.assertIn('SetupIconFile=..\\..\\assets\\LangTint.ico', self.iss)
    def test_10_uninstall_icon(self): self.assertIn('UninstallDisplayIcon={app}\\LangTint.ico', self.iss)
    def test_11_license_page(self): self.assertIn('LicenseFile=..\\..\\LICENSE', self.iss)
    def test_12_english_language(self): self.assertIn('Name: "english"', self.iss)
    def test_13_russian_language(self): self.assertIn('Name: "russian"', self.iss)
    def test_14_autostart_is_user_choice(self): self.assertIn('Name: "autostart"', self.iss)
    def test_15_old_run_value_cleared(self): self.assertRegex(self.iss, r'ValueName: "LangTint"; Flags: deletevalue')
    def test_16_new_run_value_is_explicit_run(self): self.assertIn('LangTint.exe"" --run', self.iss)
    def test_17_run_value_removed_on_uninstall(self): self.assertIn('Flags: uninsdeletevalue', self.iss)
    def test_18_launch_now_is_optional(self): self.assertIn('Flags: postinstall nowait skipifsilent', self.iss)
    def test_19_preflight_exe_not_installed(self): self.assertIn('DestName: "LangTint-preflight.exe"; Flags: dontcopy', self.iss)
    def test_20_preflight_manifest_not_installed(self): self.assertIn('LangTint-preflight.exe.manifest"; Flags: dontcopy', self.iss)
    def test_21_prepare_to_install_gate_exists(self): self.assertIn('function PrepareToInstall', self.iss)
    def test_22_preflight_runs_non_mutating_selftest(self): self.assertIn("'--self-test --report \"'", self.iss)
    def test_23_selftest_precedes_stop(self): self.assertLess(self.iss.index('SelfTestParams'), self.iss.index('StopParams'))
    def test_24_stop_after_preflight(self): self.assertIn("'--stop --report \"'", self.iss)
    def test_25_preflight_failure_aborts(self): self.assertGreaterEqual(self.iss.count('Result := FmtMessage'), 2)
    def test_26_stop_failure_aborts(self): self.assertGreaterEqual(self.iss.count('Result := ExpandConstant(\'{cm:StopFailed}\')'), 2)
    def test_27_legacy_folder_deleted_postinstall_only(self): self.assertIn('if CurStep = ssPostInstall then', self.iss)
    def test_28_no_install_delete_before_success(self): self.assertNotIn('[InstallDelete]', self.iss)
    def test_29_uninstall_uses_stop_not_self_delete(self):
        sec = self.iss.split('[UninstallRun]',1)[1].split('[UninstallDelete]',1)[0]
        self.assertIn('--stop', sec); self.assertNotIn('--uninstall', sec)
    def test_30_uninstall_cleans_legacy_folder(self): self.assertIn('[UninstallDelete]', self.iss)
    def test_31_no_powershell_runtime(self): self.assertNotIn('powershell', self.iss.lower())
    def test_32_no_installer_network_download(self): self.assertNotIn('DownloadTemporaryFile', self.iss)
    def test_33_no_http_url(self): self.assertNotIn('http://', self.iss.lower())
    def test_34_runtime_install_path_matches_setup(self): self.assertIn(r'installFolder = `Programs\LangTint`', self.app)
    def test_35_runtime_stop_mode_supported(self): self.assertIn('"--stop"', self.mode)
    def test_36_stop_preserves_autostart(self): self.assertIn('AUTOSTART_PRESERVED=YES', self.app)
    def test_37_stop_restores_taskbar(self): self.assertIn('restoreTaskbarNormal', self.app[self.app.index('func stopAndRestore'):self.app.index('func uninstall')])
    def test_38_stop_restores_cursors(self): self.assertIn('restoreSystemCursors', self.app[self.app.index('func stopAndRestore'):self.app.index('func uninstall')])
    def test_39_no_args_never_run_mode(self): self.assertIn('return "--interactive"', self.mode)
    def test_40_runtime_requires_explicit_run(self): self.assertIn('"--run"', self.mode)
    def test_41_manifest_as_invoker(self): self.assertIn('requestedExecutionLevel level="asInvoker"', self.manifest)
    def test_42_manifest_x64(self): self.assertIn('processorArchitecture="amd64"', self.manifest)
    def test_43_manifest_version(self): self.assertIn('version="1.7.0.0"', self.manifest)
    def test_44_icon_has_many_sizes(self):
        b=(ROOT/'assets/LangTint.ico').read_bytes(); self.assertEqual(b[:4], b'\0\0\1\0'); self.assertGreaterEqual(struct.unpack_from('<H', b, 4)[0], 8)
    def test_45_icon_has_256_entry(self):
        b=(ROOT/'assets/LangTint.ico').read_bytes(); n=struct.unpack_from('<H', b, 4)[0]; widths=[b[6+i*16] for i in range(n)]; self.assertIn(0, widths)
    def test_46_publisher_is_author(self): self.assertIn('#define MyAppPublisher "JetSmileOK"', self.iss)
    def test_47_product_url_is_current_repo(self): self.assertIn('https://github.com/JetSmileOK/LangTint', self.iss)
    def test_48_support_url_is_issues(self): self.assertIn('/LangTint/issues', self.iss)
    def test_49_update_url_is_releases(self): self.assertIn('/LangTint/releases', self.iss)
    def test_50_setup_logging_enabled(self): self.assertIn('SetupLogging=yes', self.iss)
    def test_51_no_restart_requested(self): self.assertIn('RestartIfNeededByRun=no', self.iss)
    def test_52_no_app_restart_magic(self): self.assertIn('RestartApplications=no', self.iss)
    def test_53_close_applications_enabled(self): self.assertIn('CloseApplications=yes', self.iss)
    def test_54_preflight_report_persists_on_failure(self): self.assertIn(r'{localappdata}\LangTint\InstallerPreflight.txt', self.iss)
    def test_55_selftest_does_not_use_sendinput(self):
        winapi=(ROOT/'source/winapi_windows.go').read_text(encoding='utf-8')
        self.assertNotIn('procSendInput', winapi); self.assertNotIn('SendInput.Call', winapi)
    def test_56_no_overlay_window_api(self): self.assertNotIn('SetLayeredWindowAttributes', self.app)
    def test_57_no_permanent_ticker(self): self.assertNotIn('time.NewTicker', self.app)
    def test_58_failure_matrix_present(self): self.assertTrue((ROOT/'source/failure_matrix_test.go').is_file())
    def test_59_hardening_logic_present(self): self.assertTrue((ROOT/'source/hardening_logic.go').is_file())
    def test_60_no_source_fragments_directory(self): self.assertFalse((ROOT/'tools/internal/source_fragments').exists())

if __name__ == '__main__': unittest.main()
