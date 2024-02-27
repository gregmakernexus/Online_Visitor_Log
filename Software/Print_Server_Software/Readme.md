# Print_Server

The print_server is a stand alone program that reads the Visitor Log database and prints badges on a label printer.  This program is written in go, and will run on any operating system that supports glables-qt.  It has been tested on:

## OS Compatability

'''
1. Ubuntu 22.04
2. Raspbian 64 bit Bookworm
3. Raspbian 32 bit Bookworm
4. Mac OS
'''

## Printer Compatibility (drivers)

'''
1. Dymo Labelwriter Turbo.  No issues.
2. Brother QL-800.  Raspbian drivers are compiled for Arm 32 bit.  Works on Bookworm 32 bit
'''

## Installation

Raspberry PI, must run the 32 bit version of the Rasbian due to issues with Brother driver.  Obtain the software from github.com.  There is a install.sh script that will install dependencies.  Since Raspberry Pi is an ARM processor, the source for glabels-qt must be recompiled during installation.     