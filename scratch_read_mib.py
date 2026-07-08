import pandas as pd
import sys

def search_excel(file_path, keywords):
    try:
        # Read all sheets
        xls = pd.ExcelFile(file_path)
        for sheet_name in xls.sheet_names:
            df = pd.read_excel(xls, sheet_name=sheet_name)
            
            # Print columns to understand structure
            print(f"--- Sheet: {sheet_name} ---")
            print(f"Columns: {df.columns.tolist()}")
            
            # Search across all string columns for keywords
            for index, row in df.iterrows():
                row_str = " ".join([str(val).lower() for val in row.values])
                for kw in keywords:
                    if kw.lower() in row_str:
                        print(f"Found '{kw}' in row {index}: {row.to_dict()}")
                        break # Go to next row if one keyword is found
                        
    except Exception as e:
        print(f"Error reading excel: {e}")

if __name__ == "__main__":
    file_path = "/workspace/mibs/NetEngine 8000 M14, M8 and M4 V800R023C00SPC500 MIB Reference.xlsx"
    keywords = ["pppoe", "active session", "subscriber", "online user"]
    search_excel(file_path, keywords)
