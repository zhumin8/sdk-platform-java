from typing import List, Optional, Dict


class DeepCopyRegexItem:
    def __init__(self, source: str, dest: str):
        self.source = source
        self.dest = dest

    def to_dict(self):
        return {
            "source": self.source,
            "dest": self.dest,
        }


class OwlbotYamlAdditionRemove:
    def __init__(
        self,
        deep_copy_regex: Optional[List[DeepCopyRegexItem]] = None,
        deep_remove_regex: Optional[List[str]] = None,
        deep_preserve_regex: Optional[List[str]] = None,
    ):
        self.deep_copy_regex = deep_copy_regex
        self.deep_remove_regex = deep_remove_regex
        self.deep_preserve_regex = deep_preserve_regex

    def to_dict(self):
        data = {}
        if self.deep_copy_regex:
            data["deep_copy_regex"] = [item.to_dict() for item in self.deep_copy_regex]
        if self.deep_remove_regex:
            data["deep_remove_regex"] = self.deep_remove_regex
        if self.deep_preserve_regex:
            data["deep_preserve_regex"] = self.deep_preserve_regex
        return data


class OwlbotYamlConfig:
    def __init__(
        self,
        addition: Optional[OwlbotYamlAdditionRemove] = None,
        remove: Optional[OwlbotYamlAdditionRemove] = None,
    ):
        self.addition = addition
        self.remove = remove

    def to_dict(self):
        data = {}
        if self.addition:
            data["addition"] = self.addition.to_dict()
        if self.remove:
            data["remove"] = self.remove.to_dict()
        return data
