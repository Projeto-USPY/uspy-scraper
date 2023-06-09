# USPY 🕵️ - Scraper

This is the official repository for the [USPY](https://uspy.me) Scraper utility! Here you can find how to run the application youself and some brief explanation on the code repository.

&nbsp;

## What does it do?

---

All data inside USPY is not merely collected by hand. This utility was created to collect subject, offering and professor data from various sources, curating them and inserting them into the database that is used by the backend

This tool leverages the power of Go routines to collect data as fast as possible and integrate it into the database in a timely manner.

The repository is divided into two different services: worker and scheduler.

The main hero here is the worker, whereas the scheduler is simply used to divide requests such that each is executed in a different Cloud Run environment. This is done for two main reasons:

1. Cloud Scheduler has a small limit on the number of cron jobs per Billing Account, whereas Cloud Run can have thousands of concurrent executions without going over the Free Tier. That way, with one request we are able to map each institute data collection to one execution.
2. Our goal is not to slow down USP services! We are not aware of how JupiterWeb handles distributed load, but by separating requests, we collect data in smaller chunks.


&nbsp;

## Running

---

The easiest way to run this is by using docker-compose. After installing docker & docker-compose, simply run:

`docker-compose -f ./local/docker-compose.yaml`

This spins up a router with a few endpoints:

- `/update?instituteCodes=1,55,3,...`
- `/build?instituteCodes=1,2,8,...`
- `/noop?instituteCodes=1,4,7,...`
- `/sync/stats`
- `/sync/subjects`

### Noop

Noop refers to "no operation". This means all data will be collected but no database operations will be performed.

This is good to measure performance (aka time to scrape).

### Update

Collects all data and then updates database, merging origial data with new one. This does not affect subject statistics for example

### Build

**USE WITH CAUTION!**

Collects all data and then rebuilds database, overwriting original data with new one. This **does** affect subject statistics for example

### Sync

This is a self healing mechanism. It takes subject data (which can change since the scraper runs from time to time) and syncs across the database. The main endpoint is `/stats`, which re-calculates stats based on all data

This can take some time and is implemented because Firestore does not provide many aggregation operators.

&nbsp;

## **How to contribute**
---

### **Features, requests, bug reports**

If this is the case, please submit an issue through the [contributions repository](github.com/Projeto-USPY/uspy-contributions/issues).

### **Actual code**

If you'd to directly contribute, fork this repository and create a pull request to merge on `dev` branch. Please do not submit pull requests to the main branch as they will be denied. The main branch is used for releases and we don't really push to it other than through the `deploy.sh` script.

If you'd to directly contribute, fork this repository and create a pull request to merge on `dev` branch. Please do not submit pull requests to the main branch as they will be denied. The main branch is used for releases and we don't really push to it other than through the `deploy.sh` script.

We really appreciate any contributors! This project is from USP students and for USP students! If you have any questions or would simply like to chat, contact us on Telegram @preischadt @lucsturci

