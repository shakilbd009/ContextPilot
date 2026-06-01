// Update app.d.ts to reflect the new locals shape
declare global {
  namespace App {
    interface Locals {
      ffEnableAppShell: boolean;
      isAuthenticated: boolean;
    }
    // interface Error {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }
}

export {};