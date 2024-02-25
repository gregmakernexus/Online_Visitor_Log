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
  rm -rfd $HOME/src
  sudo rm -rf /usr/local/go  
  set -e 
  mkdir $HOME/src
  cd $HOME/src
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
  #rm go1.21.6.linux-armv6l.tar.gz
  cd
  if [[ ":$PATH:" == *":/usr/local/go/bin:"* ]]; then
    echo "GO is installed"
  else
    echo "export PATH=$PATH:/usr/local/go/bin" >> .profile
    echo "GO was added to your path."
  fi
  if [[ "$GOPATH" != "$HOME/go" ]]; then
    echo GOPATH=$HOME/go >> .profile
  fi
  source ~/.profile
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
mkdir ~/.bin
if [[ ":$PATH:" == *":$HOME/.bin:"* ]]; then
  echo ".bin is installed"
else
  echo "export PATH=$HOME/.bin:$PATH" >> .profile
  echo ".bin was added to your path."
fi
cd ~/.bin
cp -f $HOME/Downloads/printserver.go printserver.go
go build printserver.go
cd
#-----------------------------------------------------
#  copy the template files and logo to the Mylabels directory
#------------------------------------------------------
mkdir ~/Mylabels
cd ~/Mylabels
cp -f $HOME/Downloads/maker_nexus_logo.png maker_nexus_logo.png  
cp -f $HOME/Downloads/template.glabels template.glabels
#-----------------------------------------------------
# Install dymo printer drivers
#----------------------------------------------------
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

 
