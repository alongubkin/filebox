import os
from filebox import shared_directories

def test_write_and_delete_file(shared_directories):
  """
   - Create a directory and write a file to it in some shared directory X.
   - Make sure the directory and the file appear correctly in all other shared directories.
   - Rename the new file in X.
   - Make sure all other shared directories contain the renamed file, and not the original file.
   - Delete the file and the directory in X.
   - Make sure all other shared directories don't have the deleted directory.
  
  X is each one of the 5 shared directories.

  This test covers the following commands:
  
    - GetFileAttributes
    - OpenFile
    - ReadFile
    - WriteFile
    - DeleteFile
    - CloseFile,
    - CreateDirectory
    - ReadDirectory
    - DeleteDirectory
    - Rename
  """

  # NOTE: The 'shared_directories' argument is a list of 5 directories that are automatically 
  #       managed by a Filebox client. This logic is defined in filebox.py.

  for i, directory in enumerate(shared_directories):
    # Create a directory
    dirname = 'mydir{}'.format(i)
    os.mkdir(os.path.join(directory, dirname))

    # Create a file inside it
    filename = 'myfile{}.txt'.format(i)
    content = 'Hello {}'.format(i)
    with open(os.path.join(directory, dirname, filename), 'w') as f:
      f.write(content)

    # Make sure all other shared directories have the correct file
    for other in shared_directories:
      assert dirname in os.listdir(other)
      assert filename in os.listdir(os.path.join(other, dirname))

      with open(os.path.join(other, dirname, filename), 'r') as f:
        assert(f.read() == content)

    # Rename the file
    os.rename(os.path.join(directory, dirname, filename), 
      os.path.join(directory, dirname, filename + '.new'))
    
    # Make sure all other shared directories have the renamed file
    for other in shared_directories:
      assert dirname in os.listdir(other)
      assert filename not in os.listdir(os.path.join(other, dirname))
      assert filename + '.new' in os.listdir(os.path.join(other, dirname))

      with open(os.path.join(other, dirname, filename + '.new'), 'r') as f:
        assert(f.read() == content)

    # Delete the file and the directory
    os.remove(os.path.join(directory, dirname, filename + '.new'))
    os.rmdir(os.path.join(directory, dirname))

    # Make sure all other shared directories don't have the deleted directory
    for other in shared_directories:
      assert(not os.path.exists(os.path.join(other, dirname)))
