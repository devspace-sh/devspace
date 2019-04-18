import os
import sys

from setuptools import setup
from setuptools.command.install import install


# from https://stackoverflow.com/questions/45150304/how-to-force-a-python-wheel-to-be-platform-specific-when-building-it # noqa
try:
    from wheel.bdist_wheel import bdist_wheel as _bdist_wheel

    class bdist_wheel(_bdist_wheel):
        def finalize_options(self):
            _bdist_wheel.finalize_options(self)
            # Mark us as not a pure python package (we have platform specific rust code)
            self.root_is_pure = False

        def get_tag(self):
            # this set's us up to build generic wheels.
            # note: we're only doing this for windows right now (causes packaging issues
            # with osx)
            if not sys.platform.startswith("win"):
                return _bdist_wheel.get_tag(self)

            python, abi, plat = _bdist_wheel.get_tag(self)
            python, abi = 'py2.py3', 'none'
            return python, abi, plat

except ImportError:
    bdist_wheel = None


try:
    import pypandoc
    long_description = pypandoc.convert_file("README.md", "rst")
except ImportError:
    long_description = ''

executable_name = "pip-tryout-devspace.exe" if sys.platform.startswith("win") else "pip-tryout-devspace"


class PostInstallCommand(install):
    """Post-installation for installation mode."""
    def run(self):
        # So ths builds the executable, and even installs it
        # but we can't install to the bin directory:
        #     https://github.com/pypa/setuptools/issues/210#issuecomment-216657975
        # take the advice from that comment, and move over after install
        install.run(self)
        source_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "bin")

        # setuptools_rust doesn't seem to let me specify a musl cross compilation target
        # so instead just build ourselves here =(.
        #if os.system("cargo build --release %s" % compile_args):
        #    raise ValueError("Failed to compile!")

        # we're going to install the py-spy executable into the scripts directory
        # but first make sure the scripts directory exists
        if not os.path.isdir(self.install_scripts):
            os.makedirs(self.install_scripts)

        source = os.path.join(source_dir, executable_name)
        target = os.path.join(self.install_scripts, executable_name)
        if os.path.isfile(target):
            os.remove(target)

        self.move_file(source, target)

setup(name='first-tryout-florian-kutz',
      version='0.1.0',
      cmdclass={'install': PostInstallCommand, 'bdist_wheel': bdist_wheel},
      zip_safe=False)
