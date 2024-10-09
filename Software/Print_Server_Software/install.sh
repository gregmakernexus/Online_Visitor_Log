#!/bin/bash
#---------------------------------------------
#  install program checks if the program needs to be 
#  rebuilt.  If so, builds and moves it to $HOME/.bin
#-----------------------------------------------
function install_program {
   source="${program_name}".go
   sourcepath=${PWD}
   if [[ ! -f "${source}" ]]; then
     echo "${source}" is not in the Report_Creator_Software directory
     exit 102
   fi
   echo building "${source}" 
   go build "${source}"
   cd "/${HOME}/.bin"
   mv -f "${sourcepath}/${program_name}" "${program_name}"
}

#-------------------------------------------------
# Get the CPU architecture (must be rasbian).
# The cpu architecture changes over time, and the version
# of Go will change over time.  This may need updating
# in the future.
#-------------------------------------------------
ARCH=$(uname -m)
#-------------------------------------------------
# if the install.sh command was executed from another
# directory, extract the path to the install.sh because
# that is where the other files will be.
#---------------------------------------------------
bash_path=$(dirname "$0")  # $0 contains the command executed
if [ "$bash_path" == "." ]; then
  bash_path=$(pwd)
fi 
echo path:$bash_path
#--------------------------------------------------
# Install GO
#-------------------------------------------------
if type "go" > /dev/null; then
   echo "go is installed"
else
  set +e 
  sudo rm -rf /usr/local/go  
  set -e 
  cd "$HOME/Downloads"
  if [ "$ARCH" == aarch64 ] || [ "$ARCH" == arm64* ]; then
    wget "https://dl.google.com/go/go1.22.5.linux-arm64.tar.gz"
    sudo tar -C /usr/local -xzf go1.22.5.linux-arm64.tar.gz
  elif  [[ "$ARCH" == armv* ]]; then
    wget https://dl.google.com/go/go1.22.5.linux-armv6l.tar.gz
    sudo tar -C /usr/local -xzf go1.22.5.linux-armv6l.tar.gz
  else
    echo "This script is intended for a Raspberry PI.  Unknown architecture:$ARCH"
    exit 3
  fi
  cd
  if [[ ":$PATH:" == *":/usr/local/go/bin:"* ]]; then
    echo "GO is installed"
  else
    echo "export PATH=$PATH:/usr/local/go/bin" >> .bashrc
    echo "GO was added to your path."
  fi
  if [[ "$GOPATH" != "$HOME/go" ]]; then
    echo GOPATH=$HOME/go >> .bashrc
  fi
  source ~/.bashrc
  go version
fi

#----------------------------------------------------
# install node and npm
#---------------------------------------------------
if type "npm" > /dev/null; then
  echo "npm is installed"
else
  curl -sL https://deb.nodesource.com/setup_18.x | sudo bash -
  sudo apt install nodejs
  node --version
  npm -- version
fi
#----------------------------------------------------
# install pm2
#----------------------------------------------------
if type "pm2" > /dev/null; then
  echo "pm2 is installed"
else
  sudo npm install pm2 -g
fi
#----------------------------------------------------
# Create the application .bin directory to hold applications
#---------------------------------------------------
cd
if [ ! -d "$HOME/.bin" ]; then
	mkdir ~/.bin
fi
if [[ ":$PATH:" == *":$HOME/.bin:"* ]]; then
  echo ".bin is installed"
else
  echo "export PATH=$HOME/.bin:$PATH" >> .bashrc
  source ~/.bashrc
  echo ".bin was added to your path."
fi
#----------------------------------------------------
# Copy mp3 files to the music directory
#---------------------------------------------------
cd "$HOME/Music"
# sudo apt-get update
# sudo apt-get install libasound2-dev
# sudo apt-get install libudev-dev
pkgs='libasound2-dev libudev-dev'
install=false
for pkg in $pkgs; do
  status="$(dpkg-query -W --showformat='${db:Status-Status}' "$pkg" 2>&1)"
  if [ ! $? = 0 ] || [ ! "$status" = installed ]; then
    install=true
    break
  fi
done
if "$install"; then
  sudo apt-get update
  sudo apt-get install $pkgs
else
  echo libasound2-dev and libudev-dev are already installed
fi
cp -f "$bash_path/start_me_up.mp3" start_me_up.mp3
cp -f "$bash_path/Quartz_Alarm_Clock_Beeps.mp3" Quartz_Alarm_Clock_Beeps.mp3
#-------------------------------------------------------
#  Compile printserver.go
#-------------------------------------------------------
cd "$bash_path"/printserver
program_name="printserver"
install_program
#-------------------------------------------------------
#  Compile daily_log.go
#-------------------------------------------------------
cd "$bash_path"/../Report_Creator_Software/daily_log
program_name="daily_log"
install_program
#-------------------------------------------------------
#  Compile visitor_report.go
#-------------------------------------------------------
cd "$bash_path"/../Report_Creator_Software/visitor_report
program_name="visitor_report"
install_program
#-------------------------------------------------------
#  Compile waiver_report.go
#-------------------------------------------------------
cd "$bash_path"/../Report_Creator_Software/waiver_report
program_name="waiver_report"
install_program
#------------------------------------------------------
# install test software including selenium, and a load
# of python dependencies
#-----------------------------------------------------
cd
if [ ! -d "$HOME/test" ]; then
	mkdir ~/test
  cd ~/test
  sudo apt-get install chromium-chromedriver
  cp -f "$bash_path/printserver/requirements.txt" requirements.txt
  python3 -m venv env
  source env/bin/activate 
  pip install -r requirements.txt
  echo "python environment built."
fi
cd ~/test
cp -f "$bash_path/printserver/ovlregister.py" ovlregister.py
cp -f "$bash_path/../Report_Creator_Software/waiver_report/waiverdump.py" waiverdump.py
#---------------------------------------------------
# copy files to .bin directory
#----------------------------------------------------
cd "$HOME/.bin"
cp -f "$bash_path/../Report_Creator_Software/report_creator.sh" report_creator.sh
cp -f "$bash_path/ovlregister.sh" ovlregister.sh
cp -f "$bash_path/printserver/printserver.sh" printserver.sh
if [[ $(ls) = *printserver* ]]; then
  echo "printserver is installed"
else
  exit 202
fi
if [[ $(ls) = *ovlregister.sh* ]]; then
  chmod +x ovlregister.sh
  echo "ovlregister.sh is installed"
else
  exit 203
fi
if [[ $(ls) = *printserver.sh* ]]; then
  chmod +x printserver.sh
  echo "printserver.sh is installed"
else
  exit 204
fi
if [[ $(ls) = *report_creator.sh* ]]; then
  chmod +x report_creator.sh
  echo "report_creator.sh is installed"
else
  exit 204
fi
#-------------------------------------------------------
#  Compile printconfig.go and copy to .bin
#-------------------------------------------------------
cd "$bash_path"/printconfig
program_name="printconfig"
install_program

echo "Resetting label configuration filters in $HOME/.makerNexus/labelConfig.json"
cd "$HOME/.makerNexus"
rm labelConfig.json
#-----------------------------------------------------
#  copy the template files and logo to the Mylabels directory
#------------------------------------------------------
cd
if [ ! -d "$HOME/Mylabels" ]; then 
	mkdir "$HOME/Mylabels"
fi
cd "$HOME/Mylabels"
cp -f "$bash_path/maker_nexus_logo.png" maker_nexus_logo.png  
cp -f "$bash_path/DYMO.glabels" DYMO.glabels
cp -f "$bash_path/BROTHER.glabels" BROTHER.glabels
cp -f "$bash_path/printer.glabels" printer.glabels

#-----------------------------------------------------
# Install Brother printer driver.  Set the media 2.4" diameter
# If media is not set correctly, printer will not print
#----------------------------------------------------
cd "$HOME/Downloads"
if [[ $(lpstat -a) = *QL-800* ]]; then
  lpoptions -p QL-800 -o media=24Dia
  lpoptions -p QL-800-1 -o media=24Dia
  lpoptions -p QL-800-2 -o media=24Dia
  echo "QL-800 printer driver is installed"
else
  if  [[ "$ARCH" == armv* ]]; then
    cd $bash_path
    echo Installing Brother Drivers
    rm -rf ql800*
    wget https://download.brother.com/welcome/dlfp100534/ql800pdrv-2.1.4-0.armhf.deb
    sudo dpkg -i "ql800pdrv-2.1.4-0.armhf.deb"
    echo "To verify cups installation,open chrome.  Go to: http://localhost:631/printers"
  else
    echo "Brother Printer driver only works on 32-bit Rasbian"
    echo "Skipping driver installation"
  fi
fi
#-------------------------------------------------------------
# Install dymo printer driver
#-------------------------------------------------------------
cd
if [[ $(apt list --installed) = *printer-driver-dymo* ]]; then
	echo "printer-driver-dymo is already installed"
else
	echo "installing dymo printer drivers"
	sudo apt-get -y update
	sudo apt-get -y install cups cups-client printer-driver-dymo
fi
#-----------------------------------------------------
# Install glabels-qt 
#----------------------------------------------------
cd "$HOME/Downloads"
if type "glabels-qt" > /dev/null; then
  echo "glables-qt is installed"
else
  sudo apt install -y cmake
  sudo apt install -y qtbase5-dev libqt5svg5-dev qttools5-dev zlib1g-dev
  sudo apt install -y pkgconf libqrencode-dev
  rm -rfd glabels-qt-master
  rm -fd master.zip
  wget https://github.com/jimevins/glabels-qt/archive/master.zip
  ls -l master*
  unzip -o master.zip
  rm master.zip
  cd glabels-qt-master
  mkdir build
  cd build
  echo '*----------------------------------------------------------'
  echo '* About to compile glabels-qt.  This can take a LONG time'
  echo '* let it run until it suceeds'
  echo '*----------------------------------------------------------'  
  cmake ..
  make
  sudo make install
fi


 
