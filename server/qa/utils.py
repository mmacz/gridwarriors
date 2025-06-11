import os

def find_file_upward(filename, stop_at=".git", start_dir=None):
    if start_dir is None:
        start_dir = os.getcwd()

    current_dir = os.path.abspath(start_dir)

    while True:
        candidate_path = os.path.join(current_dir, filename)
        stop_path = os.path.join(current_dir, stop_at)

        if os.path.isfile(candidate_path):
            return candidate_path

        if os.path.exists(stop_path):
            break

        parent_dir = os.path.dirname(current_dir)
        if parent_dir == current_dir:
            break

        current_dir = parent_dir

    raise FileNotFoundError(f"{filename} not found (searched up to {stop_at})")
