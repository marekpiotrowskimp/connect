#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::Listener;
use tauri::Manager;

#[tauri::command]
fn get_service_port(app: tauri::AppHandle) -> Result<u16, String> {
    let data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("data dir: {}", e))?;
    let port_file = data_dir.join(".connectd.port");
    let content = std::fs::read_to_string(&port_file)
        .map_err(|e| format!("read port file: {}", e))?;
    content
        .trim()
        .parse::<u16>()
        .map_err(|e| format!("parse port: {}", e))
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_clipboard::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .invoke_handler(tauri::generate_handler![get_service_port])
        .setup(|app| {
            // Handle deep links received during app startup
            #[cfg(desktop)]
            {
                let handle = app.handle().clone();
                app.listen("deep-link://new-url", move |event| {
                    let url = event.payload();
                    if let Some(window) = handle.get_webview_window("main") {
                        let _ = window.eval(&format!(
                            "window.__DEEP_LINK__ = {url:?}; window.dispatchEvent(new CustomEvent('deep-link', {{ detail: {url:?} }}))"
                        ));
                    }
                });
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running connect-ui");
}
