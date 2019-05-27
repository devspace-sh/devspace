// We only compile for linux no need to check for other operating systems here
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <stdbool.h>

#include <netinet/tcp.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <netinet/in.h>
#include <netdb.h>

#define BUFFER_SIZE 1024
#define HOST "techslides.com"
#define PREFIX "/demos/samples/"

// Opens a new socket connection to target host and port
int socket_connect(char *host, in_port_t port)
{
    struct hostent *hp;
    struct sockaddr_in addr;
    int on = 1, sock;

    if ((hp = gethostbyname(host)) == NULL)
    {
        herror("gethostbyname");
        exit(1);
    }

    bcopy(hp->h_addr, &addr.sin_addr, hp->h_length);
    addr.sin_port = htons(port);
    addr.sin_family = AF_INET;
    sock = socket(PF_INET, SOCK_STREAM, IPPROTO_TCP);
    setsockopt(sock, IPPROTO_TCP, TCP_NODELAY, (const char *)&on, sizeof(int));

    if (sock == -1)
    {
        perror("setsockopt");
        exit(1);
    }

    if (connect(sock, (struct sockaddr *)&addr, sizeof(struct sockaddr_in)) == -1)
    {
        perror("connect");
        exit(1);
    }

    return sock;
}

int downloadVersion(char *version, char *outFile)
{
    int fd;
    char buffer[BUFFER_SIZE];

    fd = socket_connect(HOST, 80);

    const char *connection = "Connection: close\r\n";
    const char *encoding = "Accept-Encoding: identity, *;q=0\r\n\r\n";

    char header[120];
    char hostValue[120];

    sprintf(header, "GET %s%s HTTP/1.1\r\n", PREFIX, version);
    sprintf(hostValue, "Host: %s:%d\r\n", HOST, 80);

    write(fd, header, strlen(header));
    write(fd, hostValue, strlen(hostValue));
    write(fd, connection, strlen(connection));
    write(fd, encoding, strlen(encoding));

    bzero(buffer, BUFFER_SIZE);

    FILE *f;
    int bytesRead = 0;

    bool headerTrimed = false;
    char *line = NULL, *tmp = NULL;
    size_t size = 0, index = 0;

    f = fopen(outFile, "wb");

    do
    {
        bytesRead = read(fd, buffer, BUFFER_SIZE - 1);
        if (bytesRead > 0)
        {
            if (!headerTrimed)
            {
                for (int i = 0; i < bytesRead; i++)
                {
                    char ch = buffer[i];

                    /* Check if we need to expand. */
                    if (size <= index)
                    {
                        size += 1024;
                        tmp = realloc(line, size);
                        if (!tmp)
                        {
                            free(line);
                            line = NULL;
                            break;
                        }

                        line = tmp;
                    }

                    /* Actually store the thing. */
                    line[index++] = ch;

                    // Check if \r\n\r\n is in there
                    if (ch == '\n' && strstr(line, "\r\n\r\n") != NULL)
                    {
                        headerTrimed = true;
                        int index = i + 1;

                        if (index < bytesRead)
                        {
                            // Write the rest of the buffer to the file
                            if (fwrite(&buffer[index], 1, bytesRead - index, f) != (bytesRead - index))
                            {
                                perror("header trimed fwrite");
                                exit(1);
                            }
                        }

                        break;
                    }
                }
            }
            else
            {
                // Write buffer to file
                if (fwrite(&buffer[0], 1, bytesRead, f) != bytesRead)
                {
                    perror("fwrite");
                    exit(1);
                }
            }

            bzero(buffer, BUFFER_SIZE);
        }
    } while (bytesRead != 0);

    shutdown(fd, SHUT_RDWR);
    close(fd);
    fclose(f);

    return 0;
}

int main(int argc, char *argv[])
{
    if (argc < 2)
    {
        fprintf(stderr, "Usage: %s <version>\n", argv[0]);
        exit(1);
    }

    // Make sure the tmp dir exists
    mkdir("/tmp", 0777);

    char outFile[120];
    sprintf(outFile, "/tmp/%s", argv[1]);

    // Check if the file exists
    if (access(outFile, F_OK) == -1)
    {
        // Download file
        downloadVersion(argv[1], outFile);

        // Stat file
        struct stat buf;
        if (stat(outFile, &buf) == -1)
        {
            perror("stat file");
            exit(1);
        }

        // Set file to executable
        if (chmod(outFile, buf.st_mode | S_IXUSR | S_IXGRP | S_IXOTH) == -1)
        {
            perror("chmod file");
            exit(1);
        }
    }

    return 0;
}
