#!/bin/bash

echo "Running files in /app"

for file in /app/*.java; do
  echo "Compiling $file"
  javac $file
done

for test_file in /app/*Test.java; do
  echo "Running tests from $test_file"
  java -cp .:junit-4.13.2.jar:hamcrest-core-1.3.jar org.junit.runner.JUnitCore $(basename $test_file .java)
done
