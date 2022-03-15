#!/bin/bash

mvn package -T 1C -U -Dmaven.test.skip=true && java -jar target/app-1.jar
