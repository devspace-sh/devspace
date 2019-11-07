import React, { ReactNode } from "react";
import styles from "./PageLayout.module.scss";
import Header from "components/basic/PageLayout/Header/Header";
import ErrorBoundary from "components/basic/ErrorBoundary/ErrorBoundary";
import Menu from "components/basic/PageLayout/Menu/Menu";

interface Props {
  className?: string;

  heading?: ReactNode;
  children?: ReactNode;
}

const PageLayout = (props: Props) => {
  return (
    <React.Fragment>
      <div className={styles["page-layout-container"]}>
        <Menu />

        <div className={styles["app-body"]}>
          <Header />
          {props.heading}
          <div id="scroll-container" className={styles["main-content-wrapper"]}>
            <div
              className={
                props.className
                  ? styles["main-content"] + " " + props.className
                  : styles["main-content"]
              }
            >
              <ErrorBoundary>{props.children}</ErrorBoundary>
            </div>
          </div>
        </div>
      </div>
    </React.Fragment>
  );
};

export default PageLayout;
