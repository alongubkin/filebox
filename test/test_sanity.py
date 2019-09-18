import os
from filebox import shared_directories


def test_write_and_delete_file(shared_directories):
  """
  Write a file to directory A, make sure it appears correctly in all other directories,
  then delete it in directory A, and make sure it disappears in all other directories.

  A is each one of the server directory + all client directories.
  """
  for i, directory in enumerate(shared_directories):
    filename = 'myfile{}.txt'.format(i)
    content = 'Hello {}'.format(i)

    with open(os.path.join(directory, filename), 'w') as f:
      f.write(content)

    for other in shared_directories:
      with open(os.path.join(other, filename), 'r') as f:
        assert(f.read() == content)

    os.remove(os.path.join(directory, filename))

    for other in shared_directories:
      assert(not os.path.exists(os.path.join(other, filename)))
    