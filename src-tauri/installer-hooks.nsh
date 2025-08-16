!include "FileFunc.nsh"
; Rely on built-in NSIS checkbox variable set by Tauri template: $DELETE_APPDATA

!macro NSIS_HOOK_PREUNINSTALL
  ; No custom UI here; avoid double prompts. Just log the built-in checkbox state.
  DetailPrint "[PREUNINSTALL] Built-in delete checkbox: '$DeleteAppDataCheckboxState'"
!macroend

; Optional: after uninstall cleanup based on the flag
!macro NSIS_HOOK_POSTUNINSTALL
  DetailPrint "[POSTUNINSTALL] Built-in delete checkbox: '$DeleteAppDataCheckboxState'"

  ; If user checked "Delete app data", proceed
  StrCmp $DeleteAppDataCheckboxState "1" doDelete skipDelete

  doDelete:
    ; Application data path (per-user current context)
    StrCpy $0 "$LOCALAPPDATA\MightyPieRevamped"
    DetailPrint "Delete folder: $0"
    RMDir /r /REBOOTOK "$0"
    StrCpy $1 "$APPDATA\MightyPieRevamped"
    DetailPrint "Delete folder: $1"
    RMDir /r /REBOOTOK "$1"

    ; WebView2 user data (per-user current context) under identifier folder
    StrCpy $0 "$LOCALAPPDATA\io.github.rayzorblade23.mightypie-revamped\EBWebView"
    DetailPrint "Delete folder: $0"
    RMDir /r /REBOOTOK "$0"
    ; Try removing the identifier folder if now empty
    RMDir "$LOCALAPPDATA\io.github.rayzorblade23.mightypie-revamped"

    ; Also attempt to remove for all user profiles (handles elevated uninstall context)
    ; Iterate C:\Users\* and remove both Local and Roaming variants
    Push $2
    Push $3
    Push $4
    FindFirst $2 $3 "C:\Users\*"
    loop_users:
      StrCmp $3 "." next_user
      StrCmp $3 ".." next_user
      ; Ensure it's a directory
      IfFileExists "C:\Users\$3\*.*" 0 next_user
      ; Build paths
      StrCpy $4 "C:\Users\$3\AppData\Local\MightyPieRevamped"
      DetailPrint "Delete folder: $4"
      RMDir /r /REBOOTOK "$4"
      StrCpy $4 "C:\Users\$3\AppData\Roaming\MightyPieRevamped"
      DetailPrint "Delete folder: $4"
      RMDir /r /REBOOTOK "$4"
      ; WebView2 user data under identifier folder for this user
      StrCpy $4 "C:\Users\$3\AppData\Local\io.github.rayzorblade23.mightypie-revamped\EBWebView"
      DetailPrint "Delete folder: $4"
      RMDir /r /REBOOTOK "$4"
      ; Attempt to remove identifier folder if empty
      RMDir "C:\Users\$3\AppData\Local\io.github.rayzorblade23.mightypie-revamped"
    next_user:
      FindNext $2 $3
      IfErrors done_users
      Goto loop_users
    done_users:
      FindClose $2
      Pop $4
      Pop $3
      Pop $2

  skipDelete:

  ; Try to remove the identifier-based folder if it's empty (created by Tauri)
  ; This removes the visual clutter if nothing is inside.
  RMDir "$LOCALAPPDATA\io.github.rayzorblade23.mightypie-revamped"
!macroend
