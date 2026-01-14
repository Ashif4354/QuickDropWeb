const dropzone = document.getElementById('dropzone');
const fileInput = document.getElementById('file-input');
const progressContainer = document.getElementById('progress-container');
const progressBarFill = document.getElementById('progress-bar-fill');
const resultContainer = document.getElementById('result-container');
const qrImage = document.getElementById('qr-image');
const urlText = document.getElementById('url-text');
const destructedContainer = document.getElementById('destructed-container');
const resetBtn = document.getElementById('reset-btn');

let pollInterval;

// Drag & Drop Effects
function highlight() {
    dropzone.classList.add('dragover');
}
function unhighlight() {
    dropzone.classList.remove('dragover');
}

['dragenter', 'dragover'].forEach(eventName => {
    dropzone.addEventListener(eventName, (e) => {
        e.preventDefault();
        e.stopPropagation();
        highlight();
    }, false);
});

['dragleave', 'drop'].forEach(eventName => {
    dropzone.addEventListener(eventName, (e) => {
        e.preventDefault();
        e.stopPropagation();
        unhighlight();
    }, false);
});

dropzone.addEventListener('drop', handleDrop, false);
dropzone.addEventListener('click', () => fileInput.click());
fileInput.addEventListener('change', (e) => handleFiles(e.target.files));

function handleDrop(e) {
    const dt = e.dataTransfer;
    const files = dt.files;
    handleFiles(files);
}

function handleFiles(files) {
    if (files.length > 0) {
        uploadFile(files[0]);
    }
}

function uploadFile(file) {
    // UI: Show Progress
    dropzone.style.display = 'none';
    progressContainer.style.display = 'block';

    const formData = new FormData();
    formData.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/upload', true);

    xhr.upload.onprogress = function(e) {
        if (e.lengthComputable) {
            const percentComplete = (e.loaded / e.total) * 100;
            progressBarFill.style.width = percentComplete + '%';
        }
    };

    xhr.onload = function() {
        if (xhr.status === 200) {
            const response = JSON.parse(xhr.responseText);
            showResult(response);
        } else {
            alert('Upload failed!');
            resetUI();
        }
    };

    xhr.onerror = function() {
        alert('Upload failed!');
        resetUI();
    };

    xhr.send(formData);
}

function showResult(data) {
    progressContainer.style.display = 'none';
    resultContainer.style.display = 'flex'; // flex for center alignment
    
    // Set QR and URL
    // The backend can ensure full URL is returned or we construct it.
    // Assuming backend returns { "token": "uuid", "url": "http://ip:port/download/uuid", "qr": "/qr/uuid" }
    // Or just token and we build it. Let's assume backend returns "url" and "qr_url".
    
    urlText.innerText = data.url;
    qrImage.src = data.qr_url;

    // Start Polling
    startPolling(data.token);
}

function startPolling(token) {
    if (pollInterval) clearInterval(pollInterval);

    pollInterval = setInterval(() => {
        fetch(`/status/${token}`)
            .then(res => {
                if (res.status === 404) {
                    // File is gone!
                    showDestructed();
                }
            })
            .catch(err => console.error('Polling error:', err));
    }, 1000); // Check every second
}

function showDestructed() {
    clearInterval(pollInterval);
    resultContainer.style.display = 'none';
    destructedContainer.style.display = 'block';
}

function resetUI() {
    clearInterval(pollInterval);
    destructedContainer.style.display = 'none';
    resultContainer.style.display = 'none';
    progressContainer.style.display = 'none';
    dropzone.style.display = 'flex';
    progressBarFill.style.width = '0%';
    fileInput.value = ''; // clear input
}

resetBtn.addEventListener('click', resetUI);
