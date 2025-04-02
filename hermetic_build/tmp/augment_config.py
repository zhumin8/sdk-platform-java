import os
import argparse
import yaml
import difflib
from typing import List, Optional, Dict
from common.model.generation_config import GenerationConfig

def main():
    """
    Main function to parse command-line arguments and run the augment.
    """
    parser = argparse.ArgumentParser(
        description="Compare .Owlbot-hermetic.yaml files in two directories."
    )
    parser.add_argument("input_dir", help="Path to the 'input' directory.")
    parser.add_argument("original_dir", help="Path to the 'original' directory.")
    parser.add_argument(
        "-d", "--diff", action="store_true", help="Output file differences."
    )
    parser.add_argument("-c", "--config", dest="generation_config_path", help="Path to the generation config file.")


    args = parser.parse_args()

    if args.generation_config_path:
        args.generation_config_path = args.generation_config_path.strip() 
        if not os.path.isfile(args.generation_config_path):
            raise FileNotFoundError(
                f"Generation config {args.generation_config_path} does not exist."
            )
        print(f"Generation config path: {args.generation_config_path}")
        generation_config = GenerationConfig.from_yaml(args.generation_config_path)

        generation_config.write_object_to_yaml("config_original.yaml")
    # compare_owlbot_yaml_files(args.input_dir, args.original_dir, args.diff)


if __name__ == "__main__":
    main()
