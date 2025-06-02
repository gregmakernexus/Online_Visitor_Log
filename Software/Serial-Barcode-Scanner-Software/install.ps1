# Run this as Administrator

# === Config ===
$nssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
$nssmZip = "$env:TEMP\nssm.zip"
$nssmExtractPath = "$env:ProgramFiles\nssm"
$pm2ServiceName = "pm2-service"
$pm2Home = "$env:USERPROFILE\.pm2"

# Step 1: Ensure Node.js and PM2 are installed
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Node.js using Chocolatey..."
    if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
        Set-ExecutionPolicy Bypass -Scope Process -Force
        iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
    }
    choco install -y nodejs-lts
}

Write-Host "Installing PM2 globally..."
npm install -g pm2

# Step 2: Download NSSM
Write-Host "Downloading NSSM..."
Invoke-WebRequest -Uri $nssmUrl -OutFile $nssmZip

# Step 3: Extract NSSM
Write-Host "Extracting NSSM..."
Expand-Archive -Path $nssmZip -DestinationPath $nssmExtractPath -Force

# Find nssm.exe path
$nssmExe = Get-ChildItem -Recurse -Filter "nssm.exe" -Path $nssmExtractPath | Where-Object { $_.FullName -like "*win64*" } | Select-Object -First 1

if (-not $nssmExe) {
    Write-Error "NSSM executable not found."
    exit 1
}

# Step 4: Install PM2 as a Windows service via NSSM
Write-Host "Installing PM2 as a Windows Service..."
& "$($nssmExe.FullName)" install $pm2ServiceName "C:\Program Files\nodejs\node.exe" "C:\Users\$env:USERNAME\AppData\Roaming\npm\node_modules\pm2\lib\Daemon.js"

# Set PM2_HOME so PM2 knows where to find process data
& "$($nssmExe.FullName)" set $pm2ServiceName AppEnvironmentExtra "PM2_HOME=$pm2Home"

# Step 5: Start the PM2 service
Start-Service $pm2ServiceName
Write-Host "PM2 service '$pm2ServiceName' installed and started successfully."

# Step 6 (Optional): Start your app
pm2 start barcode_scanner.exe
pm2 save
