#!/bin/bash
#-------------------------------------------------
# Get the CPU architecture (must be rasbian).
# The cpu architecture changes over time, and the version
# of Go will change over time.  This may need updating
# in the future.
#-------------------------------------------------
ARCH=$(uname -m)
#--------------------------------------------------
# Install GO
#-------------------------------------------------
if ! type "go" > /dev/null; then
  set +e 
  sudo rm -rf /usr/local/go  
  set -e 
  cd "$HOME/Downloads"
  if  [[ "$ARCH" == aarch64* ] || [ "$ARCH" == arm64* ]]; then
    wget "https://dl.google.com/go/go1.21.6.linux-arm64.tar.gz"
    sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz
  elif  [[ "$ARCH" == armv* ]]; then
    wget https://dl.google.com/go/go1.21.6.linux-armv6l.tar.gz
    sudo tar -c /usr/local -xzf go1.21.6.linux-armv6l.tar.gz
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
  go -v
fi

#----------------------------------------------------
# install node and npm
#---------------------------------------------------
if ! type "npm" > /dev/null; then
  curl -sL https://deb.nodesource.com/setup_18.x | sudo bash -
  sudo apt install nodejs
  node --version
  npm -- version
fi
#----------------------------------------------------
# install pm2
#----------------------------------------------------
if ! type "pm2" > /dev/null; then
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
  echo ".bin was added to your path."
fi


bash_path=$(dirname "$0")
cd "$bash_path/printserver"
if [ ! -f "printserver.go" ]; then
   echo "printserver.go is not in the directory with install script"
   exit 100
fi
go build printserver.go
cd "$HOME/.bin"
cp -f "$bash_path/printserver/printserver" printserver
cd
#-----------------------------------------------------
#  copy the template files and logo to the Mylabels directory
#------------------------------------------------------
if [ ! -d "$HOME/Mylabels ]; then 
	mkdir "$HOME/Mylabels"
fi
cd "$HOME/Mylabels"
cp -f "$bash_path/maker_nexus_logo.png" maker_nexus_logo.png  
cp -f "$bash_path/DYMO.glabels" DYMO.glabels
cp -f "$bash_path/BROTHER.glabels" BROTHER.glabels
#-----------------------------------------------------
# Install dymo printer drivers
#----------------------------------------------------
if  [[ "$ARCH" == armv* ]]; then
    wget https://support.brother.com/g/b/downloadend.aspx?c=us&lang=en&prod=lpql800eus&os=10041&dlid=dlfp100534_000&flang=178&type3=10261
    sudo dpkg -i ql800pdrv-2.1.4-0.armhf.deb
    echo To verify cups installation,open chrome.  Go to: http://localhost:631/printers
else
    echo Brother Printer driver only works on 32-bit Rasbian
    echo Skipping driver installation
fi
sudo apt-get -y update
sudo apt-get -y install cups cups-client printer-driver-dymo
#-----------------------------------------------------
# Install glabels-qt 
#----------------------------------------------------
if ! type "glabels-qt" > /dev/null; then
  sudo apt install -y cmake
  sudo apt install -y qtbase5-dev libqt5svg5-dev qttools5-dev zlib1g-dev
  sudo apt install -y pkgconf libqrencode-dev
  cd ~/Downloads
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

 
