import subprocess
import pathlib
import re
import configparser
import sys
from packaging import version
from typing import Tuple

# Regex for finding tagged versions in github
VERSION_TAG_REGEX = re.compile(r'refs/tags/v(?P<version>[\S]+)')

MODULE_DIR = pathlib.Path(__file__).parent.parent.parent

CONFIG_PATH = MODULE_DIR / "setup.cfg"


def get_target_major_minor_from_config(
    parser: configparser.ConfigParser
) -> Tuple[int, int]:
    """Gets the target major and minor version from the setup.cfg file."""
    target_version: str = parser["version"]["target"]
    target_split = target_version.split(".")

    try:
        target_major, target_minor = int(target_split[0]), int(target_split[1])
    except (ValueError, IndexError):
        error_message = "Version:Current setting in setup.cfg is not major.minor format"
        sys.stderr.write(error_message)
        raise ValueError(error_message)

    return target_major, target_minor


def get_latest_git_tagged_patch_version(
    major_target: int, minor_target: int
) -> int:
    """Gets the latest patch version for the target major and minor on github."""
    git_process = subprocess.Popen(
        ['git', 'ls-remote', '--tags'],
        cwd=str(MODULE_DIR),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    result, result_err = git_process.communicate(timeout=30)
    exit_code = git_process.wait()

    result_str = result.decode()
    result_err_str = result_err.decode()

    if exit_code != 0:
        raise RuntimeError(f"Error getting tag list from git: '{result_err_str}'")

    # parse versions into list
    patch_latest = -1
    for version_string in VERSION_TAG_REGEX.findall(result_str):
        try:
            version_parsed = version.parse(version_string)
        except version.InvalidVersion:
            continue

        major_parsed = version_parsed.release[0]
        minor_parsed = version_parsed.release[1]

        # If this version is from a different major / minor pairing, move over it.
        if major_parsed != major_target or minor_parsed != minor_target:
            continue

        # If the patch version is the highest we have found yet, remember it.
        patch_parsed = version_parsed.release[-1]
        if patch_parsed > patch_latest:
            patch_latest = patch_parsed

    return patch_latest


def main():
    # Parse the config file.
    parser = configparser.ConfigParser()
    parser.read(str(CONFIG_PATH))

    # Get the major and minor version we are targeting from the config
    target_major, target_minor = get_target_major_minor_from_config(parser)

    # Get the latest patch version that's been released from git
    latest_patch = get_latest_git_tagged_patch_version(target_major, target_minor)
    target_patch = latest_patch + 1

    # Concatenate the next patch version
    version_release = f"{target_major}.{target_minor}.{target_patch}"

    # Write the version we are releasing to the config.
    parser["version"]["release"] = version_release

    with CONFIG_PATH.open('w') as f:
        parser.write(f)

    # Set the variable through azure's logging variable mechanism
    print(f"##vso[task.setvariable variable=RELEASE_VERSION]{version_release}")


if __name__ == '__main__':
    main()
