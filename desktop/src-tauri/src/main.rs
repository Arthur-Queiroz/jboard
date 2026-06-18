// Oculta a janela de console no Windows em builds release.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    jboard_lib::run()
}
