# Online_Visitor_Log

## Project Synopsis
Maker Nexus is a 501(c)3 educational non-profit makerspace located in Sunnyvale CA.  In furtherance of its mission, Maker Nexus has staff, members, and many different types of visitors.

It is important for Maker Nexus to maintain an accurate list of all people who are currently in the building for safety and security reasons.  It is also important for Maker Nexus management to
have accurate documentation about the utilization of the facility and the equipment therein. Maker Nexus staff and members are issued RFID badges and are required to use these
badges to check-in and check-out of the facility, as well as to document their access to various shops and labs within the Maker Nexus facility.  

The Online Visitor Log helps Maker Nexus document visits to its facility by people who are neither staff nor members.  Maker Nexus visitors must fill out an online visitor form
that is accessible by scanning a posted QR code with their smart phone.  Upon submission of the online visitor form, a stick-on visitor badge is automatically printed and visitors are
required to wear the badge while inside of the Maker Nexus facility.  Visitor badges contain their own QR code and visitors check-out of the facility by scanning the QR code on their badge.
Repeat visitors (e.g. attendees to a Maker Nexus sponsored Meetup group) may keep their visitor badge and can scan the QR code on their badge to re-enter the facility without the need 
to fill out the online visitor form again.

The Online Visitor Log maintains a database of all visits (check-ins and check-outs) to Maker Nexus for management and marketing purposes.

The Online Visitor Log is "lightly integrated" with the Maker Nexus RFID system (https://github.com/makernexus/RFID_System).  The names of all visitors currently in the facility are listed
on an iframe within the RFID "Facility Display" so that all persons who are currently in the building (staff, members and visitors) are shown on this display.

## Project Components
The following are the components of the Online Visitor Log project:

- Cloud database:  Visit information is stored in a cloud hosted database.  Various cloud hosted PHP scripts provide web clients and the badge print server with access to check-in 
and checkout-out functions, as well as the Facility Display.

- Print Server:  A Raspberry Pi computer and its associated software obtains lists of new registrations from the cloud database and automatically prints stick-on badges for use by visitors.

- Smart Phones:  Visitors normally use their cell phones to scan QR codes in order the fill out the online visitor form and to check-in and check-out of Maker Nexus.  Maker Nexus staff personnel
can use their own smart phones or tablet to assist visitors who don't have their own smart phone available.

- Tablet:  An Android tablet is provided with a special app that continuously scans for QR codes.  Visitors can check out of Maker Nexus by scanning the QR code on their badge with this tablet in lieu
of launching a QR reader app on their own smart phone.

- Report Generator:  A Raspberry Pi computer and its associated software is logged into the Cloud database and queries the database for visit information necessary to produce various reports
for managment and marketing purposes.

## Repository Information
This repository is organized into folders:

### Documents
This folder contains an Overview document that fully describes this project

### Hardware
Information about the hardware used for this project is contained in this folder.

### Software
This folder contains source code and build/installation files for the software components of this project.  There are several subfolders, as follows:

#### Check_In-Out App
Source and installation files for the app that runs on the Tablet hardware.  The App is written in MIT App Inventor 2 and has been built for Android devices only.

#### Database-Server_Software
Contains PHP scripts, HTML and CSS code, Javascipt and related files for all of the cloud hosted software components of the system.

#### Print_Server_Software
Contains software and build/installation files for the visitor badge print server.  The software is written in Go (Golang).

#### Report_Creator_Software
Contains software and build/installation files for management report generation.  The software is written in Go (Golang).

