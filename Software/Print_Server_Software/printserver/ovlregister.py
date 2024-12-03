import glob
import os
import time
import traceback
import json
import re

from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from pathlib import Path


def main():
    home = Path.home()
    
    tour = "1"
    classworkshop = "2"
    event = "3"
    meetup = "4"
    camp = "5"
    volunteer = "6"
    guest = "7"
    forgotbadge = "8"
    other = "9"
    
    testData = [
                ["kelly","Yamanishi","kelly.yamanishi@comcast.net",[forgotbadge,tour]],
                ["No","Reason","no.reason@yahoo.com",[]],
                ["greg","Yamanishi","greg.yamanishi@gmail.com",[volunteer,event]],
                ["MyNameisReallyLong",  "lastnameistoolong", "MyNameisReallyLong@fakemail.com",[tour]],
                ["Kid1",  "smith", "test@test.com", [camp,event]],
                ["adult",  "smith", "adult@test.com", [classworkshop]],
	            ["adult",  "moneybags", "adult@gmail.com", [event]],
                ["Danielle", "Hollenbeck", "oliver@fakemail.com", [forgotbadge]],
	            ["Kyle",  "Geenen", "cara@fakemail.com",[forgotbadge]],
	            ["alsootfa",  "mohammed", "alsoofa@fakemail.com",[meetup]],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",[tour]],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",[guest]],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",[other]],
                ["mickey",  "mouse", "mikey@fakemail.com",[forgotbadge,meetup]],
                ["camp",  "meet-up", "visitor@fakemail.com",[camp,meetup]],
                ["James",  "Bell", "eventAndMeetup@member.com",[event,meetup]],
               ]
    
    for visitor in testData:
        s = Service('/usr/bin/chromedriver')
        driver = webdriver.Chrome(service=s)
        driver.maximize_window()
        driver.get("https://rfidsandbox.makernexuswiki.com/v2/OVLsignin.php")
        driver.find_element(By.ID, "mainNameFirst").send_keys(visitor[0])
        driver.find_element(By.ID, "mainNameLast").send_keys(visitor[1])
        driver.find_element(By.ID, "email").send_keys(visitor[2])
        print("Name:",visitor[0],visitor[1])
        for d in visitor[3]:
            print("Digit:",d)
            if d == tour:
                driver.find_element(By.NAME, "visitReason[]").click()
            else:
                driver.find_element(By.CSS_SELECTOR, ".visitreasoncheckbox:nth-child("+d+")").click()
        driver.find_element(By.NAME, "hasSignedWaiver").click()
        driver.find_element(By.ID, "submitbutton").click()
        time.sleep(1)
        driver.quit()


if __name__ == "__main__":
    main()
