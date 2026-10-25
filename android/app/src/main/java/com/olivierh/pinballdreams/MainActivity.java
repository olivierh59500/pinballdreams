package com.olivierh.pinballdreams;

import android.app.Activity;
import android.content.res.Configuration;
import android.os.Build;
import android.os.Bundle;
import android.view.MotionEvent;
import android.view.View;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;

import com.olivierh.pinballdreams.mobile.EbitenView;
import com.olivierh.pinballdreams.mobile.Mobile;

import go.Seq;

/** Android lifecycle bridge; rendering, controls and gameplay remain in Go. */
@SuppressWarnings("deprecation")
public final class MainActivity extends Activity {
    private EbitenView gameView;
    private boolean suspended;
    private int orientation;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        orientation = getResources().getConfiguration().orientation;
        Seq.setContext(getApplicationContext());
        Mobile.configure(getFilesDir().getAbsolutePath());
        final boolean diagnostics = (getApplicationInfo().flags
                & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0;
        Mobile.setDiagnosticsEnabled(diagnostics);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            WindowManager.LayoutParams attributes = getWindow().getAttributes();
            attributes.layoutInDisplayCutoutMode = WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            getWindow().setAttributes(attributes);
        }
        gameView = new EbitenView(this) {
            @Override
            public boolean onTouchEvent(MotionEvent event) {
                int action = event.getActionMasked();
                if (diagnostics && (action == MotionEvent.ACTION_DOWN || action == MotionEvent.ACTION_UP
                        || action == MotionEvent.ACTION_POINTER_DOWN || action == MotionEvent.ACTION_POINTER_UP)) {
                    int index = event.getActionIndex();
                    Mobile.recordTouch(action, event.getPointerId(index), event.getX(index), event.getY(index));
                }
                if (event.getActionMasked() == MotionEvent.ACTION_CANCEL) {
                    for (int index = 0; index < event.getPointerCount(); index++) {
                        Mobile.cancelTouch(event.getPointerId(index));
                    }
                    return true;
                }
                int actionIndex = event.getActionIndex();
                float density = getResources().getDisplayMetrics().density;
                for (int index = 0; index < event.getPointerCount(); index++) {
                    int pointerAction = index == actionIndex ? action : MotionEvent.ACTION_MOVE;
                    Mobile.updateTouch(pointerAction, event.getPointerId(index),
                            event.getX(index), event.getY(index), density);
                }
                return true;
            }
        };
        gameView.setFocusable(true);
        gameView.setFocusableInTouchMode(true);
        setContentView(gameView);
        gameView.requestFocus();
        hideSystemBars();
        if (Build.VERSION.SDK_INT >= 33) {
            getOnBackInvokedDispatcher().registerOnBackInvokedCallback(
                    android.window.OnBackInvokedDispatcher.PRIORITY_DEFAULT, Mobile::back);
        }
    }

    @Override
    public void onConfigurationChanged(Configuration configuration) {
        if (configuration.orientation != orientation) {
            orientation = configuration.orientation;
            Mobile.orientationChanged();
        }
        super.onConfigurationChanged(configuration);
        hideSystemBars();
    }

    @Override
    protected void onPause() {
        Mobile.suspend();
        if (gameView != null && !suspended) {
            gameView.suspendGame();
            suspended = true;
        }
        super.onPause();
    }

    @Override
    protected void onResume() {
        super.onResume();
        hideSystemBars();
        if (gameView != null && suspended) {
            gameView.resumeGame();
            suspended = false;
        }
    }

    @Override
    public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused) {
            hideSystemBars();
        } else {
            Mobile.suspend();
        }
    }

    @Override
    public void onBackPressed() {
        Mobile.back();
    }

    private void hideSystemBars() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            getWindow().setDecorFitsSystemWindows(false);
            WindowInsetsController controller = getWindow().getInsetsController();
            if (controller != null) {
                controller.hide(WindowInsets.Type.statusBars() | WindowInsets.Type.navigationBars());
                controller.setSystemBarsBehavior(WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        } else {
            getWindow().getDecorView().setSystemUiVisibility(
                    View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                            | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                            | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
        }
    }
}
