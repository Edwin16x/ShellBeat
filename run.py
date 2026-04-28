import sys
import os

# Añadimos 'src' al path para que Python encuentre el paquete shellbeat
sys.path.append(os.path.join(os.path.dirname(__file__), "src"))

from shellbeat.app import ShellBeat

if __name__ == "__main__":
    try:
        app = ShellBeat()
        app.run()
    except KeyboardInterrupt:
        print("\nSaliendo de ShellBeat...")
