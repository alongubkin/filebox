![FileBox](docs/logo.png =250x57)

FileBox allows remote users to work on the same directory in real-time. 

It's written in Go, and it's my project for the [Networking Workshop](https://www.openu.ac.il/courses/20588.htm) of the Open University of Israel.

## Features

* Support for multiple users in a client-server architecture
* Real-time - when a user edits a file, the changes are immediatley reflected to other users 
* The shared directory functions just like a regular directory (this is achieved through [FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace)
* All files are saved in the server's disk
* Cross platform (Windows, Linux, macOS)

## Design

In this section, I will explain some of the design decisions I made in the project.

### FUSE

FUSE (Filesystem in Userspace) is an API that allows user-mode programs to create virtual directories. 

When the operating system or a third-party program want to execute an operation in the virtual directory (e.g create file, delete directory, list files), the user-mode program that created the virtual directory (using FUSE) gets a callback that can handle the operation in any way it wants. In this way, it's possible to create filesystems that aren't backed by the disk (e.g: memory file systems). 

We will use FUSE in order to build a **networked** file system. When FileBox receives a callback such as "Create MyFile.txt", it communicates with the server and sends a request to create MyFile.txt. It then waits for the server's response, and notifies FUSE whether the operation succeeded or not.

I chose to write FileBox in Go because of the *cross platform requirement*. Currently, the only FUSE library that supports all major operating systems (Windows, Linux, macOS) is [cgofuse](https://github.com/billziss-gh/cgofuse), and it's written in Go. 

**FileBox supports the following operations:**

* Create + delete files
* Create + delete directories
* Get file attributes
* Read and write files 
* List files in a directory
* Rename

To simplify the solution, FileBox doesn't support symlinks or permissions (chmod / chown). 

### Network Protocol

Before designing FileBox's network protocol, I chose to take a look at some other protocols that are used for providing shared access to files over the network. 

The most prominent one was [Server Message Block (SMB)](https://wiki.wireshark.org/SMB2), which is used when you access a network path in Windows such as: `\\MomPC\Share`. An important feature of SMB is that it allows to send multiple requests before getting any response. Matching responses to requests is done by giving each command a sequence number, or ID.

This feature is important for performance because FUSE allows to execute multiple operations in parallel on the virtual directory.  

### Modules

## Installation

### Requirements

## Network Protocol Specification

### Header

All messages in FileBox's protocol start with the following header:

    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |                              Magic                            |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |            Flags              |         Header Length         |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |                           Command ID                          |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |                           Data Size                           |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+


Remarks:
 * All integers are little endian.
 * The magic is always 0xFB00FB01. 
 * Currently, the only flag available is the *Response Flag*, which is set if this message is a response (which was sent by the server). Otherwise, it is a request from the client to the server.
 * 
