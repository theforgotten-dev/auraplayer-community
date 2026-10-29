#define AppName "Aura Player"
#define AppPublisher "Aura"
#define AppExeName "aura-native.exe"

#ifndef AppVersion
  #define AppVersion "1.2.3"
#endif

#ifndef SourceDir
  #error SourceDir must point to the directory containing aura-native.exe
#endif

[Setup]
#ifdef SmokeTest
AppId={{A30BC8B7-C6EE-4D84-B6EB-64093E42AA37}
#else
AppId={{49C3D748-0C18-4FEA-83CA-87923184ECA6}}
#endif
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={localappdata}\Programs\Aura Music Player
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
OutputDir=..\dist
OutputBaseFilename=Aura-Player-Setup-{#AppVersion}
VersionInfoVersion={#AppVersion}
VersionInfoDescription=Aura Player - Audio, video y subtitulos
SetupIconFile=..\build\branding\aura.ico
ChangesAssociations=yes
MinVersion=10.0.17763
CloseApplications=yes
UninstallDisplayIcon={app}\{#AppExeName}
#ifdef SmokeTest
Compression=none
#else
Compression=lzma2
#endif
SolidCompression=yes
WizardStyle=modern
LicenseFile={#SourceDir}\legal\EULA.es.txt

[Languages]
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"; LicenseFile: "{#SourceDir}\legal\EULA.es.txt"
Name: "english"; MessagesFile: "compiler:Default.isl"; LicenseFile: "{#SourceDir}\legal\EULA.en.txt"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"
Name: "fileassoc"; Description: "Agregar Aura a Abrir con para audio y video"; GroupDescription: "Opciones adicionales:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\libmpv-2.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\LICENSE-Aura.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\LEEME.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\FORMATS.json"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\licenses\*"; DestDir: "{app}\licenses"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#SourceDir}\legal\*"; DestDir: "{app}\legal"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Aura Player"; Filename: "{app}\{#AppExeName}"
Name: "{autodesktop}\Aura Player"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Registry]
#include SourceDir + "\file-associations.iss"
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio"; ValueType: string; ValueName: ""; ValueData: "Aura media file"; Flags: uninsdeletekey; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\{#AppExeName},0"; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#AppExeName}"" ""%1"""; Tasks: fileassoc

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Iniciar Aura Player"; Flags: postinstall nowait skipifsilent

[Code]
function InitializeSetup(): Boolean;
begin
  Result := True;
  // Silent deployment must not silently bypass the agreement page.
  // This flag is an administrator assertion, not proof of end-user assent.
  if WizardSilent and (ExpandConstant('{param:ACCEPTEULA|}') <> '2026-10-24.2') then
  begin
    Log('Silent setup requires prior EULA acceptance and /ACCEPTEULA=2026-10-24.2');
    Result := False;
  end;
end;
