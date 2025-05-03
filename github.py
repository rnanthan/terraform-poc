#!/usr/bin/env python3
"""
GitHub Branch Manager Class

This module provides a class to create branches and upload JSON files to GitHub repositories.
"""

import os
import json
import base64
from github import Github


class GitHubBranchManager:
    """
    A class to manage GitHub branches and file uploads.
    """

    def __init__(self, token=None):
        """
        Initialize the GitHub Branch Manager.

        Args:
            token (str, optional): GitHub personal access token. If not provided,
                                  it will try to get it from GITHUB_TOKEN environment variable.
        """
        # Get token from parameter or environment variable
        self.token = token or os.environ.get("GITHUB_TOKEN")
        if not self.token:
            raise ValueError(
                "GitHub token is required. Provide it as a parameter or set GITHUB_TOKEN environment variable.")

        # Initialize GitHub client
        self.github = Github(self.token)

    def create_branch(self, repo_name, branch_name, base_branch="main"):
        """
        Create a new branch in the specified repository.

        Args:
            repo_name (str): Repository name in format "username/repo"
            branch_name (str): Name of the new branch to create
            base_branch (str): Name of the branch to base the new branch on

        Returns:
            bool: True if branch was created or already exists, False if error occurred
        """
        try:
            # Get the repository
            repo = self.github.get_repo(repo_name)

            # Get the base branch reference
            base_ref = repo.get_git_ref(f"heads/{base_branch}")
            base_sha = base_ref.object.sha

            try:
                # Check if branch already exists
                repo.get_git_ref(f"heads/{branch_name}")
                print(f"Branch '{branch_name}' already exists.")
            except:
                # Create new branch
                repo.create_git_ref(ref=f"refs/heads/{branch_name}", sha=base_sha)
                print(f"Created new branch: {branch_name}")

            return True

        except Exception as e:
            print(f"Error creating branch: {str(e)}")
            return False

    def upload_json_file(self, repo_name, branch_name, file_path, json_data, commit_message):
        """
        Upload a JSON file to the specified branch in the repository.

        Args:
            repo_name (str): Repository name in format "username/repo"
            branch_name (str): Name of the branch to upload to
            file_path (str): Path where the JSON file should be created in the repository
            json_data (dict): Python dictionary containing the JSON data to upload
            commit_message (str): Commit message for the file upload

        Returns:
            str: URL of the uploaded file, or None if an error occurred
        """
        try:
            # Get the repository
            repo = self.github.get_repo(repo_name)

            # Convert JSON data to formatted string
            json_content = json.dumps(json_data, indent=2)

            # Encode file content
            content_bytes = json_content.encode("utf-8")
            encoded_content = base64.b64encode(content_bytes).decode("utf-8")

            # Check if file already exists in the branch
            try:
                existing_file = repo.get_contents(file_path, ref=branch_name)
                # File exists, update it
                repo.update_file(
                    path=file_path,
                    message=commit_message,
                    content=encoded_content,
                    sha=existing_file.sha,
                    branch=branch_name
                )
                print(f"Updated JSON file {file_path} in branch {branch_name}")
            except:
                # File doesn't exist, create it
                repo.create_file(
                    path=file_path,
                    message=commit_message,
                    content=encoded_content,
                    branch=branch_name
                )
                print(f"Created JSON file {file_path} in branch {branch_name}")

            # Return URL to the file in the branch
            file_url = f"https://github.com/{repo_name}/blob/{branch_name}/{file_path}"
            return file_url

        except Exception as e:
            print(f"Error uploading file: {str(e)}")
            return None

    def create_branch_and_upload_json(self, repo_name, branch_name, base_branch,
                                      file_path, json_data, commit_message):
        """
        Create a new branch and upload a JSON file to it in one operation.

        Args:
            repo_name (str): Repository name in format "username/repo"
            branch_name (str): Name of the new branch to create
            base_branch (str): Name of the branch to base the new branch on
            file_path (str): Path where the JSON file should be created in the repository
            json_data (dict): Python dictionary containing the JSON data to upload
            commit_message (str): Commit message for the file upload

        Returns:
            str: URL of the uploaded file, or None if an error occurred
        """
        # Create the branch first
        if self.create_branch(repo_name, branch_name, base_branch):
            # Then upload the file
            return self.upload_json_file(repo_name, branch_name, file_path, json_data, commit_message)
        return None


def interactive_mode():
    """Run the GitHub Branch Manager in interactive mode."""
    # Get GitHub token
    github_token = os.environ.get("GITHUB_TOKEN")
    if not github_token:
        github_token = input("Enter your GitHub Personal Access Token: ")

    try:
        # Create the manager
        manager = GitHubBranchManager(github_token)

        # Get repository and branch information
        repo_name = input("Enter repository name (username/repo): ")
        branch_name = input("Enter new branch name: ")
        base_branch = input("Enter base branch name [main]: ") or "main"

        # Get JSON file path
        file_path = input("Enter JSON file path in repository (e.g. data/config.json): ")

        # Create JSON data
        print("Enter JSON data as key-value pairs (enter empty line to finish):")
        json_data = {}
        while True:
            line = input("Enter key:value (or empty line to finish): ")
            if not line:
                break
            try:
                key, value = line.split(":", 1)
                json_data[key.strip()] = value.strip()
            except ValueError:
                print("Invalid format. Use 'key:value'")

        # Get commit message
        commit_message = input("Enter commit message: ")

        # Create branch and upload file
        file_url = manager.create_branch_and_upload_json(
            repo_name,
            branch_name,
            base_branch,
            file_path,
            json_data,
            commit_message
        )

        if file_url:
            print(f"Success! JSON file uploaded to: {file_url}")
        else:
            print("Operation failed.")

    except Exception as e:
        print(f"Error: {str(e)}")


def example_usage():
    """Example of using the GitHubBranchManager programmatically."""
    # Create a manager instance
    manager = GitHubBranchManager("your_github_token")

    # Repository details
    repo_name = "your_username/your_repo"
    branch_name = "feature-branch"
    base_branch = "main"

    # JSON file details
    file_path = "config/settings.json"
    json_data = {
        "name": "Project Config",
        "version": "1.0.0",
        "environment": "development",
        "features": {
            "login": True,
            "payment": False
        }
    }
    commit_message = "Add configuration settings"

    # Upload the JSON file
    file_url = manager.create_branch_and_upload_json(
        repo_name,
        branch_name,
        base_branch,
        file_path,
        json_data,
        commit_message
    )

    if file_url:
        print(f"Success! JSON file uploaded to: {file_url}")
    else:
        print("Operation failed.")


if __name__ == "__main__":
    interactive_mode()

    # Uncomment to run the example usage instead
    # example_usage()