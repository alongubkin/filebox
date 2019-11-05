import os
import tempfile
import pytest
import contextlib
import subprocess
import socket
import time
import platform
import glob
import uuid

FILEBOX_BASE_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..')
FILEBOX_TEST_PORT = 8763
CLIENTS = 5

def get_filebox_executable(exe_type):
  ARCH = {
    'i386': '386',
    'x86_64': 'amd64',
    'AMD64': 'amd64',
    'x86': '386',
  }

  exe_filter = '{}-{}*-{}*'.format(
    exe_type, platform.system().lower(), ARCH[platform.machine()])

  matches = glob.glob(os.path.join(FILEBOX_BASE_PATH, 'build', exe_filter)) + \
    glob.glob(os.path.join('/build', exe_filter))
  return matches[0]


def check_socket(host, port):
  with contextlib.closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
    if sock.connect_ex((host, port)) == 0:
      return True
    else: 
      return False


@contextlib.contextmanager
def filebox_server():
  assert(not check_socket('localhost', FILEBOX_TEST_PORT))

  with tempfile.TemporaryDirectory() as server_directory:
    server_process = subprocess.Popen([
      get_filebox_executable('filebox-server'),
      '--path', server_directory,
      '--port', str(FILEBOX_TEST_PORT),
      '--verbose',
    ])

    try:
      # Wait until the port is open
      while not check_socket('localhost', FILEBOX_TEST_PORT):
        time.sleep(0.1)

      yield server_directory
    finally:
      server_process.kill()


@contextlib.contextmanager
def filebox_client():
  assert(check_socket('localhost', FILEBOX_TEST_PORT))

  client_directory = os.path.join(tempfile.gettempdir(), str(uuid.uuid4()))
  client_process = subprocess.Popen([
    get_filebox_executable('filebox-client'),
    '--mountpoint', client_directory,
    '--address', 'localhost:{}'.format(FILEBOX_TEST_PORT),
    '--verbose',
   ])

  try:
    time.sleep(0.5)
    yield client_directory
  finally:
    client_process.kill()

    # Unmount the client directory just in case
    try:
      subprocess.run(['umount', client_directory])
    except:
      pass

@pytest.fixture(scope="module")
def shared_directories():
  with contextlib.ExitStack() as stack:
    managers = [stack.enter_context(filebox_server())]
    for _ in range(CLIENTS):
      managers.append(stack.enter_context(filebox_client()))

    yield managers
