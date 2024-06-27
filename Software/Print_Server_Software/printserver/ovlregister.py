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
    camp = "4"
    volunteer = "5"
    guest = "6"
    forgotbadge = "7"
    other = "8"
    
    testData = [
                ["kelly","Yamanishi","kelly.yamanishi@comcast.net",forgotbadge],
                ["greg","Yamanishi","greg.yamanishi@gmail.com",volunteer],
                ["MyNameisReallyLong",  "lastnameistoolong", "MyNameisReallyLong@fakemail.com",tour],
                ["Kid1",  "smith", "test@test.com", camp],
                ["adult",  "smith", "adult@test.com", classworkshop],
	            ["adult",  "moneybags", "adult@gmail.com", event],
                ["Oliver", "Northwood", "oliver@fakemail.com", forgotbadge],
	            ["Cara",  "Stoneburner", "cara@fakemail.com",forgotbadge],
	            ["alsootfa",  "mohammed", "alsoofa@fakemail.com",forgotbadge],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",tour],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",guest],
                ["alsootfa",  "mohammed", "alsoofa@fakemail.com",other],
                ["mickey",  "mouse", "mikey@fakemail.com",forgotbadge]]
    
    for visitor in testData:
        s = Service('/usr/bin/chromedriver')
        driver = webdriver.Chrome(service=s)
        # driver.maximize_window()
        driver.get("https://rfidsandbox.makernexuswiki.com/v2/OVLsignin.php")
        driver.find_element(By.ID, "mainNameFirst").send_keys(visitor[0])
        driver.find_element(By.ID, "mainNameLast").send_keys(visitor[1])
        driver.find_element(By.ID, "email").send_keys(visitor[2])
        if visitor[3] == tour:
            driver.find_element(By.NAME, "visitReason[]").click()
        else:
            driver.find_element(By.CSS_SELECTOR, ".visitreasoncheckbox:nth-child("+visitor[3]+")").click()
        driver.find_element(By.NAME, "hasSignedWaiver").click()
        driver.find_element(By.ID, "submitbutton").click()
        time.sleep(1)
        driver.quit()
    exit(0)
   




if __name__ == "__main__":
    main()
