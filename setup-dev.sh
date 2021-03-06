#!/bin/bash
# SOME BASIC DEPENDENCIES
set -e
<<<<<<< HEAD
sudo apt-get install -y git libssl-dev libxml2-dev libhttp-parser-dev libssh2-1-dev curl libcurl4-gnutls-dev autoconf automake libtool git
if [[ ! $(which watchman) ]]; then
    git clone https://github.com/facebook/watchman.git
    cd watchman
    git checkout v4.9.0  # the latest stable release
    ./autogen.sh
    ./configure
    make
    sudo make install
fi
# INSTALL YARN
curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
sudo apt-get update 
sudo apt-get install yarn
=======
if [[ ! $(which ansible-playbook) ]]; then
	sudo apt-get update
	sudo apt-get install -y software-properties-common
	sudo apt-add-repository --yes --update ppa:ansible/ansible
fi
if [[ ! $(which nodejs) ]]; then
	curl -sL https://deb.nodesource.com/setup_11.x | sudo -E bash -
fi
if [[ ! $(which yarn) ]]; then
	curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
	echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
fi
sudo apt-get update
sudo apt-get install -y git libssl-dev libxml2-dev libhttp-parser-dev libssh2-1-dev curl libcurl4-gnutls-dev ansible nodejs yarn libsass-dev
# APPEARS WE DON'T NEED NODEJS-LEGACY IF USING LATEST NODE
#echo "About to install nodejs-legacy"
#if [[ $(lsb_release -sr) == '16.04' ]]; then
#	if [[ ! -f /usr/bin/env/node ]]; then
#		sudo apt-get install -y nodejs-legacy
#	fi
#fi
>>>>>>> 0b27d02b0cadde2387ec689b5b05d2822b0f414b
# INSTALL YARN DEPENDENCIES
echo "about to install yarn globals"
sudo yarn global add gulp-cli rollup typescript
echo "about to run yarn install"
yarn install
#if [[ $(lsb_release -sr) == '16.04' ]]; then
#	npm rebuild node-sass --force
#fi
# FORCE NODE_SASS REBUILD, WHICH SEEMS NECESSARY TO GET VENDOR DIRECTORY IN PLACE
# 

# INSTALL RVM
# This should be done in the bookworks playbook...
#sudo gpg --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3
#\curl -sSL https://get.rvm.io | bash -s stable

# CONFIGURE RVM AND BOOKWORKS NECESSARY PARTS SO THAT PRINTING AND JEKYLL WILL WORK
pushd tools/ansible
ansible-playbook -i 'localhost,' -c local playbook-bookworks.yml
popd

# NOW WE INSTALL RUBIES FOR CURRENT USER - BOOKWORKS ANSIBLE DOESN'T SEEM TO DO IT RIGHT
# FOR THE CURRENT USER
source /usr/local/rvm/scripts/rvm
rvm install ruby-2.4.0

cp repository_mergeheads.go src/go/src/gopkg.in/libgit2/git2go.v25/repository_mergeheads.go
cp repository_mergeheads.go src/go/src/ebw/vendor/gopkg.in/libgit2/git2go.v25


