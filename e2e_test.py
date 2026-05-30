import sys
import time
from playwright.sync_api import sync_playwright

def run():
    print("Starting e2e test...")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        
        print("Navigating to http://localhost:5173...")
        page.goto('http://localhost:5173')
        page.wait_for_load_state('networkidle')
        
        page.screenshot(path='screenshot_initial.png')
        print("Initial screenshot saved to screenshot_initial.png")
        
        try:
            # Check buttons
            print("Looking for 触发上线 button...")
            deploy_btn = page.locator('button', has_text='触发上线').first
            if deploy_btn.is_visible():
                deploy_btn.click()
                print("Clicked 触发上线 button")
                page.wait_for_timeout(2000)
                
                # Fill description
                print("Filling description...")
                textareas = page.locator('textarea')
                if textareas.count() > 0:
                    textareas.first.fill("Automated E2E test deploy with exclude files")
                    print("Filled description")
                else:
                    print("Description textarea not found")
                
                # We can try to uncheck some tree nodes if present
                # But let's keep it simple and just trigger the deploy
                
                # Confirm
                confirm_btn = page.locator('.el-dialog__footer button', has_text='确 定').first
                if not confirm_btn.is_visible():
                    confirm_btn = page.locator('.el-dialog__footer button', has_text='确认').first
                
                if confirm_btn.is_visible():
                    confirm_btn.click()
                    print("Clicked Confirm button")
                    page.wait_for_timeout(3000)
                    page.screenshot(path='screenshot_after_deploy.png')
                    print("Screenshot saved to screenshot_after_deploy.png")
                else:
                    print("Confirm button not found")
            else:
                print("触发上线 button not found")
        except Exception as e:
            print(f"Error during interaction: {e}")
            page.screenshot(path='screenshot_error.png')
            
        browser.close()
        print("Test finished.")

if __name__ == "__main__":
    run()
