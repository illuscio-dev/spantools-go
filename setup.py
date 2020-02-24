from setuptools import setup
from configparser import ConfigParser

# TO FILL OUT LIB INFO AND REQUIREMENTS: edit the [metadata] and [options] sections
# of setup.cfg


# --- SETUP SCRIPT ---
if __name__ == "__main__":

    config = ConfigParser()
    config.read("./setup.cfg")

    # run setup
    setup()
