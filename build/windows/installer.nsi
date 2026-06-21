!ifndef VERSION
!error "VERSION is required"
!endif

!ifndef SOURCE_EXE
!error "SOURCE_EXE is required"
!endif

!ifndef SOURCE_ICON
!error "SOURCE_ICON is required"
!endif

!ifndef OUT_FILE
!error "OUT_FILE is required"
!endif

Unicode true
ManifestDPIAware true
RequestExecutionLevel admin
Name "Lyn"
OutFile "${OUT_FILE}"
InstallDir "$PROGRAMFILES64\Lyn"
InstallDirRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "InstallLocation"

!include MUI2.nsh
!include LogicLib.nsh
!include WinCore.nsh

!define MUI_ABORTWARNING
!define MUI_ICON "${SOURCE_ICON}"
!define MUI_UNICON "${SOURCE_ICON}"

InstType "Typical"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  SetRegView 64
  SetShellVarContext all
FunctionEnd

Section "Lyn" SecApp
  SectionIn RO
  SetOutPath "$INSTDIR"
  DetailPrint "Stopping running Lyn processes"
  nsExec::ExecToLog 'taskkill /IM lyn.exe /F'
  File "/oname=lyn.exe" "${SOURCE_EXE}"
  File "/oname=lyn.ico" "${SOURCE_ICON}"
  CreateDirectory "$SMPROGRAMS\Lyn"
  CreateShortcut "$SMPROGRAMS\Lyn\Lyn.lnk" "$INSTDIR\lyn.exe" "" "$INSTDIR\lyn.ico"
  CreateShortcut "$SMPROGRAMS\Lyn\Uninstall Lyn.lnk" "$INSTDIR\Uninstall.exe"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "DisplayName" "Lyn"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "Publisher" "lyn-tools"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "DisplayIcon" "$INSTDIR\lyn.ico"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn" "NoRepair" 1
SectionEnd

Section "Start Lyn with Windows" SecStartup
  SectionIn 1
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "Lyn" '"$INSTDIR\lyn.exe" --start-hidden'
SectionEnd

Section "Add Lyn to system PATH" SecPath
  SectionIn 1
  nsExec::ExecToLog 'powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "$$path=[Environment]::GetEnvironmentVariable(''Path'',''Machine''); if (-not (@($$path -split '';'') -contains ''$INSTDIR'')) { [Environment]::SetEnvironmentVariable(''Path'',((@($$path -split '';'' | Where-Object { $$_ }) + ''$INSTDIR'') -join '';''),''Machine'') }"'
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment"
SectionEnd

Section "Uninstall"
  SetRegView 64
  SetShellVarContext all
  nsExec::ExecToLog 'taskkill /IM lyn.exe /F'
  DeleteRegValue HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "Lyn"
  nsExec::ExecToLog 'powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "$$path=[Environment]::GetEnvironmentVariable(''Path'',''Machine''); $$items=@($$path -split '';'' | Where-Object { $$_ -and $$_ -ne ''$INSTDIR'' }); [Environment]::SetEnvironmentVariable(''Path'',($$items -join '';''),''Machine'')"'
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment"
  Delete "$SMPROGRAMS\Lyn\Lyn.lnk"
  Delete "$SMPROGRAMS\Lyn\Uninstall Lyn.lnk"
  RMDir "$SMPROGRAMS\Lyn"
  Delete "$INSTDIR\lyn.exe"
  Delete "$INSTDIR\lyn.ico"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lyn"
SectionEnd

LangString DESC_SecApp ${LANG_ENGLISH} "Install the Lyn desktop launcher."
LangString DESC_SecStartup ${LANG_ENGLISH} "Launch Lyn hidden when Windows starts."
LangString DESC_SecPath ${LANG_ENGLISH} "Make the lyn command available in new terminals."

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} $(DESC_SecApp)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartup} $(DESC_SecStartup)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecPath} $(DESC_SecPath)
!insertmacro MUI_FUNCTION_DESCRIPTION_END
