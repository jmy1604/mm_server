rd /s /q ..\..\pkg

call build_center_server.bat
if errorlevel 1 goto exit

call build_rpc_server.bat
if errorlevel 1 goto exit

call build_game_server.bat
if errorlevel 1 goto exit

call build_login_server.bat
if errorlevel 1 goto exit
