#define AppName "Aura Music Player"
#define AppPublisher "Aura"
#define AppExeName "aura-native.exe"

#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif

#ifndef SourceDir
  #error SourceDir must point to the directory containing aura-native.exe
#endif

[Setup]
AppId={{49C3D748-0C18-4FEA-83CA-87923184ECA6}}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={localappdata}\Programs\Aura Music Player
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
OutputDir=..\dist
OutputBaseFilename=Aura-Music-Player-Setup-{#AppVersion}
UninstallDisplayIcon={app}\{#AppExeName}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"
Name: "fileassoc"; Description: "Associate supported audio files with Aura"; GroupDescription: "Additional options:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Aura Music Player"; Filename: "{app}\{#AppExeName}"
Name: "{autodesktop}\Aura Music Player"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Classes\.mp3"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.wav"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.flac"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.ogg"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.aac"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.m4a"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\.wma"; ValueType: string; ValueName: ""; ValueData: "AuraMusicPlayer.Audio"; Flags: uninsdeletevalue; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio"; ValueType: string; ValueName: ""; ValueData: "Audio file"; Flags: uninsdeletekey; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\{#AppExeName},0"; Tasks: fileassoc
Root: HKCU; Subkey: "Software\Classes\AuraMusicPlayer.Audio\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#AppExeName}"" ""%1"""; Tasks: fileassoc

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Launch Aura Music Player"; Flags: postinstall nowait skipifsilent
