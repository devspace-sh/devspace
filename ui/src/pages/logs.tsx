import React from "react";
import { withRouter, RouteComponentProps } from "react-router";
import styles from "pages/styles/logs.module.scss";
import PageLayout from "components/basic/PageLayout/PageLayout";
import withPopup, { PopupContext } from "contexts/withPopup/withPopup";

interface Props extends PopupContext, RouteComponentProps {}
interface State {}

class Logs extends React.PureComponent<Props, State> {
  render() {
    return (
      <PageLayout className={styles["spaces-component"]}>
        <div>Hello World!</div>
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(Logs));
