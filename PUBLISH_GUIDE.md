# Publishing litreader to GitHub - Quick Start Guide

This guide will help you publish your litreader project to GitHub.

## Current Status

Your project is ready to publish! All necessary files have been created:
- ✅ .gitignore (prevents unnecessary files from being committed)
- ✅ LICENSE (MIT - allows others to use your code)
- ✅ README.md (comprehensive documentation)
- ✅ CONTRIBUTING.md (guide for contributors)
- ✅ example.conf (example configuration)
- ✅ All code uses consistent naming (litreader, not txtreader)
- ✅ Tests pass
- ✅ Application builds successfully

## Step-by-Step Publishing Instructions

### 1. Create a New GitHub Repository

1. Go to https://github.com and sign in
2. Click the "+" icon in the top-right corner
3. Select "New repository"
4. Fill in the details:
   - **Repository name**: `litreader`
   - **Description**: "A terminal-based text file reader and library manager"
   - **Public** or **Private**: Choose based on your preference (Public recommended for open source)
   - **DO NOT** initialize with README, .gitignore, or license (we already have these)
5. Click "Create repository"

### 2. Initialize Git in Your litreader Directory

**IMPORTANT**: Currently, your git repository is in the parent directory (`/home/subbass/bin/claude/`). You need to create a separate repository just for litreader.

Open a terminal in the litreader directory and run:

```bash
cd /home/subbass/bin/claude/litreader

# Initialize a new git repository just for litreader
git init

# Add all files to git
git add .

# Create your first commit
git commit -m "Initial commit - litreader v2.4.5"
```

### 3. Connect to GitHub and Push

After creating the repository on GitHub, they'll show you commands. Use these (replace YOUR_USERNAME):

```bash
# Add GitHub as the remote origin
git remote add origin https://github.com/YOUR_USERNAME/litreader.git

# Push your code to GitHub
git branch -M master  # or 'main' if you prefer
git push -u origin master
```

### 4. Verify Everything Uploaded

1. Go to your repository page: `https://github.com/YOUR_USERNAME/litreader`
2. You should see:
   - README.md displayed on the main page
   - All your source files
   - LICENSE badge
   - NO compiled binary (litreader executable should be ignored)

## Common Git Commands for Future Updates

Once published, use these commands to update your repository:

```bash
# Check what files have changed
git status

# Add all changed files
git add .

# Commit with a message describing your changes
git commit -m "Description of what you changed"

# Push to GitHub
git push
```

## Alternative: GitHub Desktop

If you prefer a graphical interface:
1. Download GitHub Desktop from https://desktop.github.com/
2. Install it
3. Use "Add Existing Repository" and select your litreader folder
4. Follow the GUI prompts to publish

## Tags and Releases

To create a release on GitHub:
```bash
# Create a version tag
git tag -a v2.4.5 -m "Version 2.4.5 - Public release"

# Push the tag to GitHub
git push origin v2.4.5
```

Then on GitHub:
1. Go to your repository
2. Click "Releases"
3. Click "Draft a new release"
4. Select your tag (v2.4.5)
5. Add release notes
6. Optionally attach compiled binaries for different platforms

## Need Help?

- GitHub documentation: https://docs.github.com/en/get-started
- Git basics: https://git-scm.com/book/en/v2/Getting-Started-Git-Basics
- Or use GitHub Desktop for a simpler experience

## What's Already Protected

Your .gitignore file ensures these won't be committed:
- Compiled binary (litreader)
- Test binaries
- Your personal .claude/ settings
- IDE files
- OS-specific files

You're all set! 🚀
