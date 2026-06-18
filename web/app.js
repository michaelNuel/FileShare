// --- 1. DOM Elements ---
const tabSendBtn = document.getElementById('tabSendBtn');
const tabReceiveBtn = document.getElementById('tabReceiveBtn');
const sendScreen = document.getElementById('sendScreen');
const receiveScreen = document.getElementById('receiveScreen');
const dropZone = document.getElementById('dropZone');
const fileInput = document.getElementById('fileInput');
const fileDetails = document.getElementById('fileDetails');
const fileNameText = document.getElementById('fileNameText');
const fileSizeText = document.getElementById('fileSizeText');
const cancelFileBtn = document.getElementById('cancelFileBtn');
const shareBtn = document.getElementById('shareBtn');
const codeContainer = document.getElementById('codeContainer');
const shareCodeText = document.getElementById('shareCodeText');
const copyCodeBtn = document.getElementById('copyCodeBtn');
const codeInput = document.getElementById('codeInput');
const downloadBtn = document.getElementById('downloadBtn');
const progressScreen = document.getElementById('progressScreen');
const statusTitle = document.getElementById('statusTitle');
const statusPctText = document.getElementById('statusPctText');
const progressBarFill = document.getElementById('progressBarFill');
const bytesCounter = document.getElementById('bytesCounter');
const transferSpeed = document.getElementById('transferSpeed');

// --- 2. Global Variables ---
let selectedFile = null;
let websocket = null;
const CHUNK_SIZE = 64 * 1024; // 64KB chunks (optimal for WebSockets)

// --- 3. Tab Navigation ---
tabSendBtn.addEventListener('click', () => {
    tabSendBtn.classList.add('active');
    tabReceiveBtn.classList.remove('active');
    sendScreen.classList.remove('hidden');
    receiveScreen.classList.add('hidden');
    resetState();
});

tabReceiveBtn.addEventListener('click', () => {
    tabReceiveBtn.classList.add('active');
    tabSendBtn.classList.remove('active');
    receiveScreen.classList.remove('hidden');
    sendScreen.classList.add('hidden');
    resetState();
});

// --- 4. Drag & Drop File Selection ---
dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    if (e.dataTransfer.files.length > 0) {
        handleFileSelect(e.dataTransfer.files[0]);
    }
});

fileInput.addEventListener('change', () => {
    if (fileInput.files.length > 0) {
        handleFileSelect(fileInput.files[0]);
    }
});

function handleFileSelect(file) {
    selectedFile = file;
    fileNameText.textContent = file.name;
    fileSizeText.textContent = formatBytes(file.size);
    dropZone.classList.add('hidden');
    fileDetails.classList.remove('hidden');
}

cancelFileBtn.addEventListener('click', () => {
    resetState();
});

// Helper: Format file size
function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// --- 5. Reset UI ---
function resetState() {
    selectedFile = null;
    if (websocket) {
        websocket.close();
    }
    fileInput.value = '';
    dropZone.classList.remove('hidden');
    fileDetails.classList.add('hidden');
    codeContainer.classList.add('hidden');
    progressScreen.classList.add('hidden');
    codeInput.value = '';
}

// --- 6. Hashing Helper (Web Crypto API) ---
async function calculateHash(file) {
    const arrayBuffer = await file.arrayBuffer();
    const hashBuffer = await crypto.subtle.digest("SHA-256", arrayBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

// --- 7. WebSocket Handlers ---

// Connects to the same host we loaded the website from
function getWebSocketUrl() {
    const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
    return protocol + window.location.host + '/ws';
}

// --- SENDER FLOW ---
shareBtn.addEventListener('click', async () => {
    if (!selectedFile) return;

    shareBtn.disabled = true;
    shareBtn.textContent = 'Calculating Hash...';

    try {
        const hash = await calculateHash(selectedFile);
        shareBtn.classList.add('hidden');

        websocket = new WebSocket(getWebSocketUrl());

        websocket.onopen = () => {
            // Send SHARE command
            const cmd = `SHARE ${selectedFile.name} ${selectedFile.size} ${hash}`;
            websocket.send(cmd);
        };

        websocket.onmessage = async (event) => {
            const msg = event.data;
            const parts = msg.split(' ');

            if (parts[0] === 'CODE') {
                shareCodeText.textContent = parts[1];
                codeContainer.classList.remove('hidden');
            } else if (parts[0] === 'START') {
                codeContainer.classList.add('hidden');
                progressScreen.classList.remove('hidden');
                statusTitle.textContent = 'Streaming file chunks...';
                await streamFile();
            } else if (parts[0].startsWith('ERROR')) {
                alert('Server error: ' + msg);
                resetState();
            }
        };

        websocket.onerror = (err) => {
            console.error('WebSocket Error:', err);
            alert('Connection failed');
            resetState();
        };

    } catch (err) {
        console.error(err);
        alert('Failed to hash file');
        resetState();
    }
});

// Slices the file and streams it chunk-by-chunk
async function streamFile() {
    let offset = 0;
    const totalSize = selectedFile.size;

    while (offset < totalSize) {
        // Slice the chunk
        const slice = selectedFile.slice(offset, offset + CHUNK_SIZE);
        const buffer = await slice.arrayBuffer();

        // Send binary data
        websocket.send(buffer);

        offset += buffer.byteLength;

        // Update progress bar
        const pct = ((offset / totalSize) * 100).toFixed(1);
        statusPctText.textContent = `${pct}%`;
        progressBarFill.style.width = `${pct}%`;
        bytesCounter.textContent = `${formatBytes(offset)} / ${formatBytes(totalSize)}`;
    }

    statusTitle.textContent = 'Transfer Complete!';
    transferSpeed.textContent = 'Success!';
    websocket.close();
}

// Copy Code Button
copyCodeBtn.addEventListener('click', () => {
    navigator.clipboard.writeText(shareCodeText.textContent);
    alert('Code copied to clipboard!');
});

// --- RECEIVER FLOW ---
downloadBtn.addEventListener('click', () => {
    const code = codeInput.value.trim();
    if (code.length !== 6) {
        alert('Please enter a valid 6-digit code');
        return;
    }

    downloadBtn.disabled = true;
    websocket = new WebSocket(getWebSocketUrl());

    let targetFileName = '';
    let targetFileSize = 0;
    let targetHash = '';
    let receivedBytes = 0;
    let chunks = [];

    websocket.onopen = () => {
        // Send DOWNLOAD command
        websocket.send(`DOWNLOAD ${code}`);
    };

    websocket.onmessage = async (event) => {
        const data = event.data;

        // WebSockets can receive TEXT messages (handshake) or BINARY messages (file chunks)
        if (typeof data === 'string') {
            const parts = data.split(' ');

            if (parts[0] === 'METADATA') {
                targetFileName = parts[1];
                targetFileSize = parseInt(parts[2], 10);
                targetHash = parts[3];

                // Show progress screen
                receiveScreen.classList.add('hidden');
                progressScreen.classList.remove('hidden');
                statusTitle.textContent = `Downloading ${targetFileName}...`;

                // Signal ready to receive binary bytes
                websocket.send('READY');
            } else if (data.startsWith('ERROR')) {
                alert('Server Error: ' + data);
                resetState();
            }
        } else {
            // Binary chunk received!
            // Browser WebSockets return binary payloads as a Blob
            const arrayBuffer = await data.arrayBuffer();
            chunks.push(arrayBuffer);
            receivedBytes += arrayBuffer.byteLength;

            // Update progress bar
            const pct = ((receivedBytes / targetFileSize) * 100).toFixed(1);
            statusPctText.textContent = `${pct}%`;
            progressBarFill.style.width = `${pct}%`;
            bytesCounter.textContent = `${formatBytes(receivedBytes)} / ${formatBytes(targetFileSize)}`;
        }
    };

    websocket.onclose = async () => {
        if (receivedBytes === 0) {
            downloadBtn.disabled = false;
            return;
        }

        statusTitle.textContent = 'Verifying Integrity (SHA-256)...';
        
        // Merge all ArrayBuffer chunks into a single Blob
        const fileBlob = new Blob(chunks);
        
        // Calculate hash of downloaded file
        const calculatedHash = await calculateHash(fileBlob);

        if (calculatedHash === targetHash) {
            statusTitle.textContent = 'Download Complete!';
            transferSpeed.textContent = 'Verified!';

            // Trigger browser save dialog
            const downloadUrl = URL.createObjectURL(fileBlob);
            const a = document.createElement('a');
            a.href = downloadUrl;
            a.download = targetFileName;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(downloadUrl);
        } else {
            statusTitle.textContent = 'Verification Failed!';
            transferSpeed.textContent = 'Corrupt File!';
            alert('Warning: SHA-256 checksum mismatch. File is corrupted!');
        }
        
        downloadBtn.disabled = false;
    };

    websocket.onerror = (err) => {
        console.error('WebSocket Error:', err);
        alert('Connection failed');
        resetState();
    };
});