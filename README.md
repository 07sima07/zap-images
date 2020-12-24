# Zap images

To run on windows just open the exe file from the repository and fill in the parameters.
On windows you will probably get a notification from windows defender, just bypass it by clicking on "more info" under the notification and press the "Run anyway" button. The source code is available in the file main.go, you can check it and compile code on your pc if you are worried about possible viruses.

Parameters for loading pictures:
1. Database name - enter the name of your database downloaded from phpmyadmin (example: 7zap_skoda)
2. Database user - enter your database user name (example: root)
3. Database password - enter your database password (example: password123)
4. Database server - enter the location address of the database (default: localhost)
5. Directory for images - press path to images folder, the folder will be created in the same place where download script is located (default: images)
6. Threads - enter the number of threads (default: 2). I recommend to use no more than 10 threads (the target site may fail from the load).

In the table "group_part" will appear a new column "downloaded_image", it contains the path to the image.
Images downloaded with errors will be ignored.

![console_downloader](https://i.imgur.com/eSxzyM0.png)
