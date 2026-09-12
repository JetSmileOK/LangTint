#define MyAppName "LangTint"
#define MyAppVersion "1.7.1"
#define MyAppPublisher "JetSmileOK"
#define MyAppURL "https://github.com/JetSmileOK/LangTint"
#define MyAppExeName "LangTint.exe"

[Setup]
AppId={{A73736F0-09B4-4A73-93CC-9DFF98F62050}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL=https://github.com/JetSmileOK/LangTint/issues
AppUpdatesURL=https://github.com/JetSmileOK/LangTint/releases
DefaultDirName={localappdata}\Programs\LangTint
DefaultGroupName=LangTint
DisableProgramGroupPage=yes
DisableWelcomePage=no
DisableReadyPage=no
DisableFinishedPage=no
PrivilegesRequired=lowest
SetupArchitecture=x64
ArchitecturesAllowed=x64os
MinVersion=10.0.14393
OutputDir=output
OutputBaseFilename=LangTint-Setup-x64
SetupIconFile=..\..\assets\LangTint.ico
UninstallDisplayName=LangTint
UninstallDisplayIcon={app}\LangTint.ico
Uninstallable=yes
CreateUninstallRegKey=yes
UninstallFilesDir={app}
LicenseFile=..\..\LICENSE
WizardStyle=modern dynamic
; Use plain BMP for Setup chrome. This deliberately avoids PNG/zlib decoding before the wizard opens.
WizardSmallImageFile=..\..\assets\LangTint-WizardLight.bmp
WizardSmallImageFileDynamicDark=..\..\assets\LangTint-WizardDark.bmp
Compression=lzma2/max
SolidCompression=yes
CloseApplications=yes
RestartApplications=no
RestartIfNeededByRun=no
SetupLogging=yes
UsePreviousAppDir=yes
UsePreviousLanguage=yes
VersionInfoVersion=1.7.1.0
VersionInfoCompany=JetSmileOK
VersionInfoDescription=LangTint Setup
VersionInfoProductName=LangTint
VersionInfoProductVersion=1.7.1
VersionInfoCopyright=Copyright (c) 2026 JetSmileOK

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[CustomMessages]
english.AutoStart=Start LangTint automatically with Windows
russian.AutoStart=Запускать LangTint автоматически вместе с Windows
english.LaunchNow=Launch LangTint now
russian.LaunchNow=Запустить LangTint сейчас
english.PreflightRunning=Checking Windows, Explorer taskbar and visual backend before installation...
russian.PreflightRunning=Проверяю Windows, панель Explorer и визуальный механизм перед установкой...
english.PreflightFailed=LangTint could not verify compatibility. Nothing was installed. Diagnostic file: %1
russian.PreflightFailed=LangTint не смог подтвердить совместимость. Ничего не установлено. Диагностика: %1
english.StopFailed=An existing LangTint process could not be stopped safely. Close it and run Setup again.
russian.StopFailed=Не удалось безопасно остановить работающий LangTint. Закройте его и запустите установщик снова.

[Tasks]
Name: "autostart"; Description: "{cm:AutoStart}"; Flags: checkedonce

[Files]
; Preflight copies are embedded only for PrepareToInstall and never installed.
Source: "build\LangTint.exe"; DestName: "LangTint-preflight.exe"; Flags: dontcopy noencryption
Source: "build\LangTint.exe.manifest"; DestName: "LangTint-preflight.exe.manifest"; Flags: dontcopy noencryption

; Installed product files.
Source: "build\LangTint.exe"; DestDir: "{app}"; DestName: "LangTint.exe"; Flags: ignoreversion
Source: "build\LangTint.exe.manifest"; DestDir: "{app}"; DestName: "LangTint.exe.manifest"; Flags: ignoreversion
Source: "..\..\assets\LangTint.ico"; DestDir: "{app}"; Flags: ignoreversion

[Registry]
; Always clear an old Run value first so unchecking the task really disables autostart on upgrade.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueName: "LangTint"; Flags: deletevalue
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "LangTint"; ValueData: """{app}\LangTint.exe"" --run"; Tasks: autostart; Flags: uninsdeletevalue

[Run]
Filename: "{app}\LangTint.exe"; Parameters: "--run"; Description: "{cm:LaunchNow}"; Flags: postinstall nowait skipifsilent runasoriginaluser

[UninstallRun]
Filename: "{app}\LangTint.exe"; Parameters: "--stop --report ""{tmp}\LangTint-UninstallStop.txt"""; Flags: runhidden waituntilterminated skipifdoesntexist

[UninstallDelete]
Type: filesandordirs; Name: "{localappdata}\LangTint"

[Code]
var
  PreflightReport: String;

function RunEmbeddedLangTint(const Parameters: String; var ResultCode: Integer): Boolean;
var
  ExePath: String;
begin
  ExePath := ExpandConstant('{tmp}\LangTint-preflight.exe');
  Result := Exec(ExePath, Parameters, '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
  SelfTestParams: String;
  StopParams: String;
begin
  Result := '';
  NeedsRestart := False;
  WizardForm.StatusLabel.Caption := ExpandConstant('{cm:PreflightRunning}');

  try
    ExtractTemporaryFile('LangTint-preflight.exe');
    ExtractTemporaryFile('LangTint-preflight.exe.manifest');
  except
    Result := 'Unable to extract the LangTint preflight binary.';
    Exit;
  end;

  PreflightReport := ExpandConstant('{tmp}\LangTint-InstallerPreflight.txt');
  SelfTestParams := '--self-test --report "' + PreflightReport + '"';

  if not RunEmbeddedLangTint(SelfTestParams, ResultCode) then
  begin
    Result := FmtMessage(ExpandConstant('{cm:PreflightFailed}'), [PreflightReport]);
    Exit;
  end;
  if ResultCode <> 0 then
  begin
    Result := FmtMessage(ExpandConstant('{cm:PreflightFailed}'), [PreflightReport]);
    Exit;
  end;

  // Stop an already-running v1.6/v1.7 watcher only after the non-mutating self-test passed.
  // --stop preserves the existing autostart value, so a later Setup failure does not destroy the old install.
  StopParams := '--stop --report "' + ExpandConstant('{tmp}\LangTint-stop.txt') + '"';
  if not RunEmbeddedLangTint(StopParams, ResultCode) then
  begin
    Result := ExpandConstant('{cm:StopFailed}');
    Exit;
  end;
  if ResultCode <> 0 then
  begin
    Result := ExpandConstant('{cm:StopFailed}');
    Exit;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    // The old one-click v1.6 location was %LOCALAPPDATA%\LangTint.
    // Remove it only after the new files and registry entries are installed.
    DelTree(ExpandConstant('{localappdata}\LangTint'), True, True, True);
  end;
end;
