#!/usr/bin/env node

/**
 * Enterprise GenAI Toolbox - NPM Binary Installer
 *
 * Automatically downloads and installs the correct binary for the current platform.
 * Supports: macOS (Intel/Apple Silicon), Linux (amd64/arm64), Windows (amd64)
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const os = require('os');
const zlib = require('zlib');

const PACKAGE_VERSION = require('../package.json').version;
const BINARY_NAME = 'genai-toolbox';
const GITHUB_REPO = 'sethdford/genai-toolbox-enterprise';
const RELEASE_BASE_URL = `https://github.com/${GITHUB_REPO}/releases/download`;

/**
 * Detect current platform and architecture
 */
function getPlatformInfo() {
  const platform = os.platform();
  const arch = os.arch();

  let osName, archName, ext = '';

  // Map Node.js platform to Go GOOS
  switch (platform) {
    case 'darwin':
      osName = 'darwin';
      break;
    case 'linux':
      osName = 'linux';
      break;
    case 'win32':
      osName = 'windows';
      ext = '.exe';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  // Map Node.js arch to Go GOARCH
  switch (arch) {
    case 'x64':
      archName = 'amd64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  return { osName, archName, ext };
}

/**
 * Download file from URL
 */
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading from: ${url}`);

    const file = fs.createWriteStream(dest);

    https.get(url, {
      headers: { 'User-Agent': 'genai-toolbox-installer' }
    }, (response) => {
      // Handle redirects
      if (response.statusCode === 302 || response.statusCode === 301) {
        file.close();
        fs.unlinkSync(dest);
        return downloadFile(response.headers.location, dest)
          .then(resolve)
          .catch(reject);
      }

      if (response.statusCode !== 200) {
        file.close();
        fs.unlinkSync(dest);
        return reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
      }

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      file.close();
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

/**
 * Extract tar.gz file
 */
function extractTarGz(archivePath, destDir) {
  return new Promise((resolve, reject) => {
    const tar = require('tar');

    fs.createReadStream(archivePath)
      .pipe(zlib.createGunzip())
      .pipe(tar.extract({ cwd: destDir }))
      .on('finish', resolve)
      .on('error', reject);
  });
}

/**
 * Extract zip file (for Windows)
 */
function extractZip(archivePath, destDir) {
  // For Windows, we'll use a simple approach
  // In production, consider using a library like 'adm-zip' or 'unzipper'
  const AdmZip = require('adm-zip');
  const zip = new AdmZip(archivePath);
  zip.extractAllTo(destDir, true);
}

/**
 * Main installation function
 */
async function install() {
  console.log('');
  console.log('🚀 Installing Enterprise GenAI Toolbox...');
  console.log('');

  try {
    // Detect platform
    const { osName, archName, ext } = getPlatformInfo();
    console.log(`Platform detected: ${osName}/${archName}`);

    // Determine download URL and archive format
    const isWindows = osName === 'windows';
    const archiveExt = isWindows ? 'zip' : 'tar.gz';
    const archiveName = `${BINARY_NAME}-${osName}-${archName}.${archiveExt}`;
    const downloadUrl = `${RELEASE_BASE_URL}/v${PACKAGE_VERSION}/${archiveName}`;

    // Create bin directory
    const binDir = path.join(__dirname, '..', 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Download archive
    const archivePath = path.join(binDir, archiveName);
    console.log('Downloading binary...');
    await downloadFile(downloadUrl, archivePath);
    console.log('✓ Download complete');

    // Extract archive
    console.log('Extracting archive...');
    if (isWindows) {
      extractZip(archivePath, binDir);
    } else {
      await extractTarGz(archivePath, binDir);
    }
    console.log('✓ Extraction complete');

    // Make binary executable (Unix-like systems)
    const binaryPath = path.join(binDir, BINARY_NAME + ext);
    if (!isWindows) {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Clean up archive
    fs.unlinkSync(archivePath);

    // Create wrapper script
    const wrapperPath = path.join(binDir, `${BINARY_NAME}.js`);
    const wrapperContent = `#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');

const binaryPath = path.join(__dirname, '${BINARY_NAME}${ext}');
const args = process.argv.slice(2);

const child = spawn(binaryPath, args, { stdio: 'inherit' });

child.on('exit', (code) => {
  process.exit(code);
});
`;
    fs.writeFileSync(wrapperPath, wrapperContent);
    fs.chmodSync(wrapperPath, 0o755);

    console.log('');
    console.log('✅ Enterprise GenAI Toolbox installed successfully!');
    console.log('');
    console.log('Run with: npx genai-toolbox --help');
    console.log('Or: npx toolbox --help');
    console.log('');

  } catch (error) {
    console.error('');
    console.error('❌ Installation failed:', error.message);
    console.error('');
    console.error('Please try manual installation:');
    console.error(`  https://github.com/${GITHUB_REPO}/releases/tag/v${PACKAGE_VERSION}`);
    console.error('');
    process.exit(1);
  }
}

// Run installation if called directly
if (require.main === module) {
  install();
}

module.exports = { install, getPlatformInfo };
