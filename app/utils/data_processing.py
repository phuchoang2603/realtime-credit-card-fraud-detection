import sys

import pandas as pd
from fastapi.encoders import jsonable_encoder

sys.path.append("..")
from schema import HouseInfo


def format_input_data(data: HouseInfo):
    """Format the input data to a prediction data structure

    Args:
        data (HouseInfo): Information about a house

    Returns:
        A Pandas DataFrame: Convert the input data into a Pandas DataFrame
    """
    return pd.DataFrame(jsonable_encoder(data), index=[0])
