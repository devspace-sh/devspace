import React from 'react';
import styles from './GitHubButton.module.scss';
import CloseButton from 'components/basic/IconButton/CloseButton/CloseButton';
import IconGitHub from 'images/icon-github.svg';
import GitHubStarButton from 'react-github-btn';

interface Props {}

const LOCAL_STORAGE_KEY_HIDE_GITHUB_BUTTON = 'devspace-hide-gh-button';
const LOCAL_STORAGE_KEY_FLATTEN_GITHUB_BUTTON = 'devspace-flatten-gh-button';

class GitHubButton extends React.PureComponent<Props> {
    render() {
        const hideGithubButtonUntil = localStorage.getItem(LOCAL_STORAGE_KEY_HIDE_GITHUB_BUTTON);
        const flattenGithubButtonUntil = localStorage.getItem(LOCAL_STORAGE_KEY_FLATTEN_GITHUB_BUTTON);

        if (!hideGithubButtonUntil || parseInt(hideGithubButtonUntil) < Date.now()){
            let githubButtonClasses = styles.githubIcon;

            if (!flattenGithubButtonUntil || parseInt(flattenGithubButtonUntil) < Date.now()) {
                githubButtonClasses += ' ' + styles.highlighted;
            }

            return (
                <div
                className={styles.github}
                onMouseEnter={() => {
                    localStorage.setItem(
                    LOCAL_STORAGE_KEY_FLATTEN_GITHUB_BUTTON,
                    (Date.now() + 30 * 24 * 60 * 60 * 1000).toString()
                    );
                    this.forceUpdate();
                }}
                >
                    <CloseButton
                        className={styles['close']}
                        filter={false}
                        white={true}
                        onClick={() => {
                        localStorage.setItem(LOCAL_STORAGE_KEY_HIDE_GITHUB_BUTTON, (Date.now() + 30 * 24 * 60 * 60 * 1000).toString());
                        this.forceUpdate();
                        }}
                    />
                    <div className={githubButtonClasses}>
                        <img src={IconGitHub} />
                    </div>
                    <div className={styles.githubDetails}>
                        <h2>Support DevSpace on GitHub!</h2>
                        <p>Nothing motivates us more to build great features.</p>
                        <div className={styles.githubProjects}>
                            <div>
                                <h3>DevSpace</h3>
                                <div className="star-button">
                                    <GitHubStarButton
                                        href="https://github.com/loft-sh/devspace"
                                        data-size="large"
                                        data-show-count={true}
                                        aria-label="Star devspace-cloud/devspace on GitHub"
                                    >
                                        Star
                                    </GitHubStarButton>
                                </div>
                            </div>
                            <div>
                                <h3>DevSpace Cloud</h3>
                                <div className="star-button">
                                    <GitHubStarButton
                                        href="https://github.com/loft-sh/devspace-cloud"
                                        data-size="large"
                                        data-show-count={true}
                                        aria-label="Star devspace-cloud/devspace-cloud on GitHub"
                                    >
                                        Star
                                    </GitHubStarButton>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            );
        }
        return ""
    }
}

export default GitHubButton;
