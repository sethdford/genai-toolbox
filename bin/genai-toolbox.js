#!/usr/bin/env node

/**
 * Enterprise GenAI Toolbox - NPM Wrapper Script
 *
 * This wrapper script launches the native binary for the current platform.
 * The actual binary is downloaded during npm install via scripts/install.js
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

// Determine the binary name and extension
const isWindows = os.platform() === 'win32';
const binaryExt = isWindows ? '.exe' : '';
const binaryName = `genai-toolbox${binaryExt}`;
const binaryPath = path.join(__dirname, binaryName);

// Check if binary exists
if (!fs.existsSync(binaryPath)) {
  console.error('');
  console.error('❌ Binary not found!');
  console.error('');
  console.error('The native binary was not downloaded during installation.');
  console.error('This usually means:');
  console.error('  1. Your platform is not supported');
  console.error('  2. The release assets are not available');
  console.error('  3. Network error during installation');
  console.error('');
  console.error('Supported platforms:');
  console.error('  - macOS (Intel and Apple Silicon)');
  console.error('  - Linux (x64 and arm64)');
  console.error('  - Windows (x64)');
  console.error('');
  console.error('Try manual installation:');
  console.error('  https://github.com/sethdford/genai-toolbox-enterprise/releases/latest');
  console.error('');
  process.exit(1);
}

// Forward all arguments to the binary
const args = process.argv.slice(2);
const child = spawn(binaryPath, args, {
  stdio: 'inherit',
  windowsHide: false
});

// Forward exit code
child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code || 0);
  }
});

// Handle errors
child.on('error', (err) => {
  console.error('');
  console.error('❌ Failed to launch genai-toolbox:', err.message);
  console.error('');
  process.exit(1);
});
