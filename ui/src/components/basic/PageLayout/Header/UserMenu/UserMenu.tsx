import React from "react";
import styles from "./UserMenu.module.scss";
import withPopup, { PopupContext } from "contexts/withPopup/withPopup";
import { withRouter, RouteComponentProps } from "react-router-dom";
import ErrorBoundary from "components/basic/ErrorBoundary/ErrorBoundary";
import Tooltip from "components/basic/Tooltip/Tooltip";
import CustomNavLink from "components/basic/CustomNavLink/CustomNavLink";

interface Props extends PopupContext, RouteComponentProps {}
interface State {
  login?: string;
  picture?: string;
  menuOpen: boolean;
}

class UserMenu extends React.PureComponent<Props & RouteComponentProps, State> {
  state: State = {
    menuOpen: false
  };

  render() {
    return (
      <ErrorBoundary>
        <Tooltip
          className={styles.tooltipcontainer}
          text="Quickstart"
          position="bottom"
        >
          <CustomNavLink
            to={"/guides/start"}
            className={styles.link + " " + styles.docs}
            activeClassName={styles["selected"]}
          />
        </Tooltip>
      </ErrorBoundary>
    );
  }
}

export default withRouter(withPopup(UserMenu));
